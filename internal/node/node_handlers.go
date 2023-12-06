package node

import (
	"github.com/gweebg/mcast/internal/packets"
	"github.com/gweebg/mcast/internal/streamer"
	"github.com/gweebg/mcast/internal/utils"
	"log"
	"net"
	"net/netip"
	"strconv"
)

func (n *Node) NodeOnDiscovery(incoming packets.Packet, conn net.Conn) {
	/*
		did I handle the packet ? (check requestId in db)
		  no -> am I streaming the content ?
			yes -> answer with YES
			no -> do I have neighbours aside from source ?
				yes -> flood, wait for response and answer with it
				no -> answer with NO
		  yes -> ignore
	*/

	remote := conn.RemoteAddr().String()
	requestId := incoming.Header.RequestId
	contentName := incoming.Payload.ContentName

	log.Printf("(handling %v) received 'DISC' packet for content '%v'\n", remote, contentName)

	defer func() {
		n.HandledRequests.Set(requestId, true)
		log.Printf("(handling %v) request '%v' set to handled\n", remote, requestId)
	}() // set packet as handled

	if n.HandledRequests.IsHandled(requestId) {
		log.Printf("(handling %v) incoming packet refers to a duplicate request\n", remote)
		Reply(
			packets.Miss(requestId, contentName),
			conn,
		)
		log.Printf("(handling %v) send 'MISS', reason 'duplicate'\n", remote)
		return
	} // packet was already handled

	if n.RelayPool.IsStreaming(contentName) {
		log.Printf("(handling %v) i am streaming the content '%v'\n", requestId, contentName)
		Reply(
			packets.Found(requestId, contentName, n.Self.SelfIp),
			conn,
		)
		log.Printf("(handling %v) send 'FOUND' for content '%v'\n", remote, contentName)
		return
	} // I'm already streaming the content

	addrPort, err := netip.ParseAddrPort(conn.RemoteAddr().String())
	utils.Check(err)

	if len(streamer.FilterNeighbour(addrPort, n.Self.Neighbours)) > 0 { // do I have neighbours ?

		log.Printf("(handling %v) flooding the neighbours, looking for '%v'\n", remote, contentName)
		incoming.Header.Hops++

		// response, holds the answer resulted from the flooding
		response, _ := n.Flooder.Flood(incoming, addrPort)

		if response.Header.Flags.OnlyHasFlag(packets.FOUND) {

			log.Printf("(%v) found streaming source\n", requestId)

			err := n.PositiveRequests.Set(requestId, response.Header.Source)
			utils.Check(err)

			response.Header.Source = n.Self.SelfIp
			log.Printf("(handling %v) received positive response from flooding, pos=%v\n", remote, n.Self.SelfIp)

		} else {
			log.Printf("(handling %v) received negative response from flooding\n", remote)
		}

		Reply(response, conn)
		log.Printf("(handling %v) sent packet from flood for content '%v'", remote, contentName)

	} else { // I don't have neighbours
		log.Printf("(handling %v) list of neighbours is empty\n", remote)
		Reply(
			packets.Miss(requestId, contentName),
			conn,
		)
		log.Printf("(handling %v) sent 'MISS' packet, reason 'no neighbours'\n", remote)
		return
	}
}

func (n *Node) NodeOnStream(incoming packets.Packet, conn net.Conn) {

	/*
		am I streaming the content ?
			yes -> reply with PORT, wait for OK, add source address to relay

			no -> did I receive a positive answer for this request id before ?
				yes -> forward packet to that address & wait for PORT
				no -> reply with NOEXISTS
	*/

	remote := conn.RemoteAddr().String()
	requestId := incoming.Header.RequestId
	contentName := incoming.Payload.ContentName

	log.Printf("(handling %v) received 'STREAM' packet for content '%v'\n", remote, contentName)

	defer func() {
		utils.CloseConnection(conn, remote)
		log.Printf("(handling %v) closing connection, reason 'finished handling stream'\n", remote)
	}()

	if n.RelayPool.IsStreaming(contentName) {

		log.Printf("(handling %v) i am streaming the content '%v'\n", remote, contentName)

		relay, exists := n.RelayPool.GetRelay(contentName) // get relay for contentName
		if !exists {
			log.Fatalf("(handling %v) relay does not exist for content '%v'\n", remote, contentName)
		}

		nextAddress := utils.ReplacePortFromAddressString(remote, relay.Port)
		Reply(packets.Port(requestId, contentName, nextAddress), conn) // send addr:port
		log.Printf("(handling %v) sent 'PORT' packet, addr=%v\n", remote, nextAddress)

		err := relay.Add(nextAddress) // add client to relay
		utils.Check(err)

		log.Printf("(handling %v) added client address '%v' to the relay for content '%v'\n", remote, nextAddress, contentName)
		return
	}

	source, exists := n.PositiveRequests.CheckAndGet(requestId)
	if exists {

		log.Printf("(handling %v) previouly received 'FOUND' packet from '%v', following until source\n", remote, source)

		incoming.Header.Hops++
		response, err := Follow(incoming, source)

		if err != nil || response.Header.Flags.OnlyHasFlag(packets.MISS) {
			log.Printf("(handling %v) received 'MISS' packet from the follow\n", remote)
			Reply(
				packets.Miss(requestId, contentName),
				conn,
			)
			log.Printf("(handling %v) sending 'MISS' packet, reason 'content does not exist'\n", remote)
			return
		}

		if response.Header.Flags.OnlyHasFlag(packets.PORT) {

			log.Printf("(handling %v) received 'PORT' packet from the follow\n", remote)

			relayPort := strconv.FormatUint(n.RelayPool.NextPort(), 10)
			relay := streamer.NewRelay(contentName, response.Payload.Port, relayPort)
			log.Printf("(handling %v) created new relay for content '%v' at port '%v'\n", remote, contentName, relay.Port)

			nextAddress := utils.ReplacePortFromAddressString(remote, relay.Port)
			err := relay.Add(nextAddress)
			log.Printf("(handling %v) added address '%v' to relay for '%v'\n", remote, nextAddress, contentName)

			err = n.RelayPool.SetRelay(contentName, relay)
			utils.Check(err)
			log.Printf("(handling %v) added relay for '%v' to the relay pool\n", remote, contentName)

			go relay.Loop()
			log.Printf("(handling %v) started relay for content '%v'\n", remote, contentName)

			Reply(packets.Port(requestId, contentName, nextAddress), conn)

			log.Printf("(handling %v) sent 'PORT' packet, addr=%v", remote, nextAddress)
			return
		}

	} else {
		log.Printf("(handling %v) no positive 'FOUND' packets\n", remote)
		Reply(
			packets.Miss(requestId, contentName),
			conn,
		)
		log.Printf("(handling %v) sent 'MISS' packet, reason 'no positive'\n", remote)
		return
	}

}

func (n *Node) NodeOnTeardown(incoming packets.Packet, conn net.Conn) {

	/*
		am I streaming the content ?
		no  -> return
		yes -> am I streaming the content to >1 addresses ?
			yes -> stop streaming to incoming source
			no  -> delete the relay all together
	*/

	contentName := incoming.Payload.ContentName
	requestId := incoming.Header.RequestId
	remote := conn.RemoteAddr().String()

	defer func() {
		utils.CloseConnection(conn, remote)
		log.Printf("(handling %v) closing connection, reason 'finished handling teardown'\n", remote)
	}()

	log.Printf("(handling %v) received a packet 'TEARDOWN' for content '%v'\n", remote, contentName)

	if !n.RelayPool.IsStreaming(contentName) {
		log.Printf("(handling %v) cannot teardown since i'm not streaming content '%v'\n", remote, contentName)
		return
	}

	relay, _ := n.RelayPool.GetRelay(contentName)
	if len(relay.Addresses) > 1 {
		log.Printf("(handling %v) stopped transmitting content '%v'\n", remote, contentName)
		relay.Remove(remote)
		return
	}

	log.Printf("(handling %v) deleted relay for content '%v'\n", remote, contentName)

	err := relay.Stop()
	utils.Check(err)

	n.RelayPool.DeleteRelay(contentName)

	nextHop, exists := n.PositiveRequests.CheckAndGet(requestId)
	if !exists {
		log.Fatalf("(handling %v) there should exists a positive answer, but it doesn't\n", remote)
	}

	log.Printf("(handling %v) found positive next hop, following...\n", remote)

	nextPacket := packets.Teardown(requestId, contentName, n.Self.SelfIp)
	_, _ = Follow(nextPacket, nextHop)
}
