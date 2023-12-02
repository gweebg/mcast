package node

import (
	"github.com/gweebg/mcast/internal/packets"
	"github.com/gweebg/mcast/internal/utils"
	"log"
	"net"
	"net/netip"
	"strconv"
)

func (n *Node) OnDiscovery(incoming packets.Packet, conn net.Conn) {
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

	defer func(n *Node) {
		n.Requests.Set(requestId, true)
		log.Printf("(handling %v) request '%v' set to handled\n", remote, requestId)
	}(n) // set packet as handled

	if n.Requests.IsHandled(requestId) {
		log.Printf("(handling %v) incoming packet refers to a duplicate request\n", remote)
		reply(
			packets.Miss(requestId, contentName),
			conn,
		)
		log.Printf("(handling %v) send 'MISS', reason 'duplicate'\n", remote)
		return
	} // packet was already handled

	if n.IsStreaming(contentName) {
		log.Printf("(handling %v) i am streaming the content '%v'\n", requestId, contentName)
		reply(
			packets.Found(requestId, contentName, n.Self.SelfIp),
			conn,
		)
		log.Printf("(handling %v) send 'FOUND' for content '%v'\n", remote, contentName)
		return
	} // I'm already streaming the content

	addrPort, err := netip.ParseAddrPort(conn.RemoteAddr().String())
	utils.Check(err)

	if len(filterNeighbour(addrPort, n.Self.Neighbours)) > 0 { // do I have neighbours ?

		log.Printf("(handling %v) flooding the neighbours, looking for '%v'\n", remote, contentName)
		incoming.Header.Hops++

		// response, holds the answer resulted from the flooding
		response, _ := n.Flooder.Flood(incoming, addrPort)

		if response.Header.Flags.OnlyHasFlag(packets.FOUND) {
			log.Printf("(%v) found streaming source\n", requestId)
			n.SetPositive(requestId, response.Header.Source)
			incoming.Header.Source = n.Self.SelfIp // todo: check this
			log.Printf("(handling %v) received positive response from flooding, pos=%v\n", remote, n.Self.SelfIp)
		} else {
			log.Printf("(handling %v) received negative response from flooding\n", remote)
		}

		reply(response, conn)
		log.Printf("(handling %v) sent packet from flood for content '%v'", remote, contentName)

	} else { // I don't have neighbours
		log.Printf("(handling %v) list of neighbours is empty\n", remote)
		reply(
			packets.Miss(requestId, contentName),
			conn,
		)
		log.Printf("(handling %v) sent 'MISS' packet, reason 'no neighbours'\n", remote)
		return
	}
}

func (n *Node) OnStream(incoming packets.Packet, conn net.Conn) {

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
		utils.CloseConnection(conn, conn.RemoteAddr().String())
		log.Printf("(handling %v) closing connection, reason 'finished handling'\n", remote)
	}()

	if n.IsStreaming(contentName) {

		log.Printf("(handling %v) i am streaming the content '%v'\n", remote, contentName)

		relay, exists := n.RelayPool[contentName] // get relay for contentName
		if !exists {
			log.Fatalf("(handling %v) relay does not exist for content '%v'\n", remote, contentName)
		}

		reply(packets.Port(requestId, contentName, relay.Port), conn) // send port
		log.Printf("(handling %v) sent 'PORT' packet, port=%v\n", remote, relay.Port)

		address := utils.ReplacePortFromAddressString(remote, relay.Port)
		err := relay.Add(address) // add client to relay
		utils.Check(err)

		log.Printf("(handling %v) added client address '%v' to the relay for content '%v'\n", remote, address, contentName)
		return
	}

	source, exists := n.IsPositive(requestId)
	if exists {

		log.Printf("(handling %v) previouly received 'FOUND' packet from '%v', following until source\n", remote, source)

		incoming.Header.Hops++
		response, err := follow(incoming, source)
		if err != nil || response.Header.Flags.OnlyHasFlag(packets.MISS) {
			log.Printf("(handling %v) received 'MISS' packet from the follow\n", remote)
			reply(
				packets.Miss(requestId, contentName),
				conn,
			)
			log.Printf("(handling %v) sending 'MISS' packet, reason 'content does not exist'\n", remote)
			return
		}

		if response.Header.Flags.OnlyHasFlag(packets.PORT) {

			log.Printf("(handling %v) received 'PORT' packet from the follow\n", remote)

			videoSource := utils.ReplacePortFromAddressString("127.0.0.1:9999", response.Payload.Port)

			relayPort := strconv.FormatUint(n.NextPort(), 10)
			relay := NewRelay(contentName, videoSource, relayPort)
			log.Printf("(handling %v) created new relay for content '%v' at port '%v'\n", remote, contentName, relay.Port)

			err := n.AddRelay(contentName, relay)
			utils.Check(err)
			log.Printf("(handling %v) added address '%v' to relay for '%v'\n", remote, videoSource, contentName)

			go relay.Loop()
			log.Printf("(handling %v) started relay for content '%v'\n", remote, contentName)

			reply(packets.Port(requestId, contentName, relay.Port), conn)

			log.Printf("(handling %v) sent 'PORT' packet, port=%v", remote, relay.Port)
			return
		}

	} else {
		log.Printf("(handling %v) no positive 'FOUND' packets\n", remote)
		reply(
			packets.Miss(requestId, contentName),
			conn,
		)
		log.Printf("(handling %v) sent 'MISS' packet, reason 'no positive'\n", remote)
		return
	}

}

func reply(response packets.Packet, conn net.Conn) {

	enc, err := response.Encode()
	utils.Check(err)

	_, err = conn.Write(enc)
	if err != nil {
		log.Printf("(handlers.go) could not write to '%v'\n", conn.RemoteAddr().String())
	}

}

func follow(packet packets.Packet, destination string) (packets.Packet, error) {

	// connection setup
	conn, err := net.Dial("tcp", destination)
	if err != nil {
		log.Printf("(handlers.go) cannot connect to '%v' via tcp\n", destination)
		return packets.Packet{}, err
	}
	defer utils.CloseConnection(conn, destination)

	// packet encoding
	buffer, err := packet.Encode()
	utils.Check(err)

	// sending packet to dest
	_, err = conn.Write(buffer)
	if err != nil {
		log.Printf("(handlers.go) cannot write packet to '%v'\n", destination)
		return packets.Packet{}, err
	}

	// reading and decoding the response
	responseBuffer := make([]byte, 1024)
	n, err := conn.Read(responseBuffer)
	if err != nil {
		log.Printf("(handlers.go) cannot read packet from '%v'\n", destination)
		return packets.Packet{}, err
	}

	resp, err := packets.DecodePacket(responseBuffer[:n])
	utils.Check(err)

	return resp, nil
}
