package node

import (
	"github.com/gweebg/mcast/internal/packets"
	"github.com/gweebg/mcast/internal/streamer"
	"github.com/gweebg/mcast/internal/utils"
	"log"
	"net"
	"strconv"
)

func (n *Node) RendOnDiscovery(incoming packets.Packet, conn net.Conn) {
	/*
		did I handle this request already ?
			yes -> reply with miss
			no  -> does the content exist within the known servers ?
				yes -> reply with found
				no ->  reply with miss
	*/

	remote := conn.RemoteAddr().String()
	requestId := incoming.Header.RequestId
	contentName := incoming.Payload.ContentName

	defer func() {
		n.HandledRequests.Set(requestId, true)
		log.Printf("(handling %v) request '%v' handled\n", remote, contentName)
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

	if n.Servers.ContentExists(contentName) { // this content is available

		log.Printf("(handling %v) content '%v' is available for streaming\n", remote, contentName)
		Reply(
			packets.Found(requestId, contentName, n.Self.SelfIp),
			conn,
		)
		log.Printf("(handling %v) sent 'FOUND' with source address as '%v'\n", remote, n.Self.SelfIp)
		return

	} else { // this content is not available in any server

		log.Printf("(handling %v) content '%v' is not available for streaming\n", remote, contentName)
		Reply(
			packets.Miss(requestId, contentName),
			conn,
		)
		log.Printf("(handling %v) send 'MISS', reason 'content not found'\n", remote)
	}
}

func (n *Node) RendOnStream(incoming packets.Packet, conn net.Conn) {
	/*
		am I streaming the content ?
			yes -> reply with port and add address to corresponding relay
			no  -> select best server for the content, create and init relay, reply with port
	*/

	remote := conn.RemoteAddr().String()
	requestId := incoming.Header.RequestId
	contentName := incoming.Payload.ContentName

	log.Printf("(handling %v) received 'STREAM' packet for content '%v'\n", remote, incoming.Payload.ContentName)
	defer func() {
		utils.CloseConnection(conn, remote)
		log.Printf("(handling %v) closed connection, reason 'termination'\n", remote)
	}()

	if n.RelayPool.IsStreaming(contentName) { // if am I streaming contentName

		log.Printf("(handling %v) stream found for content '%v'\n", remote, contentName)

		relay, exists := n.RelayPool.GetRelay(contentName)
		if !exists {
			log.Fatalf("(handling %v) relay does not exist for content '%v'\n", remote, contentName)
		}

		// address to add to the relay
		nextAddress := utils.ReplacePortFromAddressString(remote, relay.Port)

		Reply(packets.Port(requestId, contentName, nextAddress), conn) // reply with streaming port
		log.Printf("(handling %v) responded with 'PORT' packet, addr=%v", remote, nextAddress)

		err := relay.Add(nextAddress) // add client to relay
		utils.Check(err)
		log.Printf("(handling %v) added address '%v' to the relay for '%v'\n", remote, nextAddress, contentName)
		return
	}

	log.Printf("(handling %v) no relay found for content '%v', asking server\n", remote, contentName)
	incoming.Header.Hops++

	// the best server to ask the stream from
	svr := n.Servers.GetBestServer(contentName)
	log.Printf("(handling %v) selected server at '%v' for the streaming of '%v'\n", remote, svr.Address, contentName)

	// stopping metrics measurement to avoid conflicts
	svr.TickerChan <- true

	// clearing out the 'pong' dangling packets
	//b := make([]byte, ReadSize)
	//_, err := svr.Conn.Read(b)

	//utils.Check(err)
	log.Printf("(metrics %v) temporarily stopped metric analysis with server\n", svr.Address)

	defer func() {
		svr.Measure()
		log.Printf("(metrics %v) restored metric analysis with server\n", svr.Address)
	}() // resuming metrics after the talk with the server

	// create request packet for the received content name
	packet := packets.Request(contentName)
	buffer, err := packets.Encode[string](packet)
	utils.Check(err)

	// send request packet to the best server
	_, err = svr.Conn.Write(buffer)
	if err != nil {
		log.Printf("(servers %v) cannot write packet 'REQ'\n", svr.Address)
		return
	}
	log.Printf("(servers %v) sent packet 'REQ' for '%v'\n", svr.Address, contentName)

	// receive and decode the response
	responseBuffer := make([]byte, ReadSize)
	s, err := svr.Conn.Read(responseBuffer)
	if err != nil {
		log.Printf("(servers %v) cannot read packet\n", svr.Address)
		return
	}

	resp, err := packets.Decode[string](responseBuffer[:s])
	utils.Check(err)

	if !resp.Header.Flag.OnlyHasFlag(packets.CSND) {
		log.Printf("(servers %v) did not receive port for stream of '%v'\n", conn.RemoteAddr().String(), contentName)
		Reply(
			packets.Miss(requestId, contentName),
			conn,
		)
		log.Printf("(handling %v) sent packet 'MISS', reason 'no port from server'\n", remote)
		return
	}

	log.Printf("(servers %v) received packet 'CSND' with addr=%v\n", svr.Address, resp.Payload)
	log.Printf("(servers %v) server is streaming '%v' at address '%v'\n", svr.Address, contentName, resp.Payload)

	// create new relay
	relayPort := strconv.FormatUint(n.RelayPool.NextPort(), 10)
	relay := streamer.NewRelay(contentName, resp.Payload, relayPort)
	log.Printf("(handling %v) created new relay for '%v', relay port is '%v'", remote, contentName, relayPort)

	// add the address of the prev node to the relay
	nextAddress := utils.ReplacePortFromAddressString(remote, relay.Port)
	err = relay.Add(nextAddress)
	utils.Check(err)
	log.Printf("(handling %v) added address '%v' to relay for '%v'\n", remote, nextAddress, contentName)

	// add relay to pool
	err = n.RelayPool.SetRelay(contentName, relay)
	utils.Check(err)
	log.Printf("(handling %v) added relay for '%v' to the pool\n", remote, contentName)

	// start the relay forwarding loop
	go relay.Loop()
	log.Printf("(handling %v) relay started transmitting '%v' with origin at '%v'\n", remote, contentName, resp.Payload)

	ok := packets.Ok()
	okResponse, err := packets.Encode[string](ok)
	utils.Check(err)

	_, err = svr.Conn.Write(okResponse)
	if err != nil {
		log.Fatalf("(servers %v) cannot reply with 'OK' to server\n", svr.Address)
	}
	log.Printf("(servers %v) sent packet 'OK'\n", svr.Address)

	svr.SetStreaming(contentName)

	Reply(packets.Port(requestId, contentName, nextAddress), conn) // reply to client in which port I'm streaming
	log.Printf("(handling %v) sent packet 'PORT', addr=%v\n", remote, nextAddress)
}

func (n *Node) RendOnTeardown(incoming packets.Packet, conn net.Conn) {

	/*
		am I streaming the content ?
		no  -> return
		yes -> am I streaming the content to >1 addresses ?
			yes -> stop streaming to incoming source
			no  -> delete the relay all together
	*/

	contentName := incoming.Payload.ContentName
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
		relay.Remove(incoming.Payload.Port)
		return
	}

	stopPacket, err := packets.Encode[string](packets.Stop(contentName))

	serv, exists := n.Servers.WhoIsStreaming(contentName)
	if !exists {
		log.Fatalf("(handling %v) server '%v' should exist, but it doesn't\n", remote, relay.Origin)
	}

	_, err = serv.Conn.Write(stopPacket)
	if err != nil {
		log.Printf("(server %v) could not send 'STOP' packet to server for content '%v'\n", serv.Address, contentName)
	}
	log.Printf("(server %v) sent packet 'STOP' to server for content '%v'\n", relay.Origin, contentName)

	log.Printf("(handling %v) deleted relay for content '%v'\n", remote, contentName)

	err = relay.Stop()
	utils.Check(err)

	n.RelayPool.DeleteRelay(contentName)
	serv.UnsetStreaming(contentName)
}
