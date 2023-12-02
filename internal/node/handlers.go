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

	requestId := incoming.Header.RequestId
	contentName := incoming.Payload.ContentName
	log.Printf("(%v) received discovery packet from '%v'\n", requestId, incoming.Header.Source)

	defer func(n *Node) {
		n.Requests.Set(requestId, true)
	}(n) // set packet as handled

	if n.Requests.IsHandled(requestId) {
		log.Printf("(%v) request id already handled, miss\n", requestId)
		reply(
			packets.Miss(requestId, contentName),
			conn,
		)
		return
	} // packet was already handled

	if n.IsStreaming(contentName) {
		log.Printf("(%v) streaming the content, found\n", requestId)
		reply(
			packets.Found(requestId, contentName, n.Self.SelfIp),
			conn,
		)
		return
	} // I'm already streaming the content

	addrPort, err := netip.ParseAddrPort(conn.RemoteAddr().String())
	utils.Check(err)

	if len(filterNeighbour(addrPort, n.Self.Neighbours)) > 0 { // do I have neighbours ?

		incoming.Header.Hops++

		// response, holds the answer resulted from the flooding
		log.Printf("(%v) flooding...\n", requestId)
		response, _ := n.Flooder.Flood(incoming, addrPort)

		if response.Header.Flags.OnlyHasFlag(packets.FOUND) {
			log.Printf("(%v) found streaming source\n", requestId)
			n.SetPositive(requestId, response.Header.Source)
			incoming.Header.Source = n.Self.SelfIp // todo: check this
		}

		reply(response, conn)
	} else { // I don't have neighbours
		log.Printf("(%v) no neighbours, miss\n", requestId)
		reply(
			packets.Miss(requestId, contentName),
			conn,
		)
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

	requestId := incoming.Header.RequestId
	contentName := incoming.Payload.ContentName
	log.Printf("(%v) received discovery packet from '%v'\n", requestId, incoming.Header.Source)

	defer utils.CloseConnection(conn, conn.RemoteAddr().String()) // todo: ???

	if n.IsStreaming(contentName) {

		relay, exists := n.RelayPool[contentName] // get relay for contentName
		if !exists {
			log.Fatalf("relay does not exist for content '%v'\n", contentName)
		}

		address := utils.ReplacePortFromAddressString(conn.RemoteAddr().String(), relay.Port)

		reply(packets.Port(requestId, contentName, relay.Port), conn) // send port

		err := relay.Add(address) // add client to relay
		utils.Check(err)

		return
	}

	source, exists := n.IsPositive(requestId)
	if exists {

		incoming.Header.Hops++
		response, err := follow(incoming, source)
		if err != nil || response.Header.Flags.OnlyHasFlag(packets.MISS) {
			reply(
				packets.Miss(requestId, contentName),
				conn,
			)
			return
		}

		if response.Header.Flags.OnlyHasFlag(packets.PORT) {

			// todo: is this the right address ?
			videoSource := utils.ReplacePortFromAddressString("127.0.0.1:0000", response.Payload.Port)

			relayPort := strconv.FormatUint(n.NextPort(), 10)
			relay := NewRelay(contentName, videoSource, relayPort)

			err := n.AddRelay(contentName, relay)
			utils.Check(err)

			go relay.Loop()

			reply(packets.Port(requestId, contentName, relay.Port), conn)
			return

		}

	} else {
		reply(
			packets.Miss(requestId, contentName),
			conn,
		)
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
