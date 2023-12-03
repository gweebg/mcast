package rendezvous

import (
	"log"
	"net"
	"strconv"

	"github.com/gweebg/mcast/internal/node"
	"github.com/gweebg/mcast/internal/packets"
	"github.com/gweebg/mcast/internal/utils"
)

func (r *Rendezvous) OnDiscovery(incoming packets.Packet, conn net.Conn) {
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
		r.Requests.Set(requestId, true)
		log.Printf("(handling %v) request '%v' handled\n", remote, contentName)
	}() // set packet as handled

	if r.Requests.IsHandled(requestId) {
		log.Printf("(handling %v) incoming packet refers to a duplicate request\n", remote)
		reply(
			packets.Miss(requestId, contentName),
			conn,
		)
		log.Printf("(handling %v) send 'MISS', reason 'duplicate'\n", remote)
		return
	} // packet was already handled

	if r.ContentExists(contentName) { // this content is available

		log.Printf("(handling %v) content '%v' is available for streaming\n", remote, contentName)
		reply(
			packets.Found(requestId, contentName, r.Address.String()),
			conn,
		)
		log.Printf("(handling %v) sent 'FOUND' with source address as '%v'\n", remote, r.Address.String())
		return

	} else { // this content is not available in any server

		log.Printf("(handling %v) content '%v' is not available for streaming\n", remote, contentName)
		reply(
			packets.Miss(requestId, contentName),
			conn,
		)
		log.Printf("(handling %v) send 'MISS', reason 'content not found'\n", remote)
	}
}

func (r *Rendezvous) OnStream(incoming packets.Packet, conn net.Conn) {
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

	if r.IsStreaming(contentName) { // if am I streaming contentName

		log.Printf("(handling %v) stream found for content '%v'\n", remote, contentName)

		relay, exists := r.RelayPool[contentName] // get correct relay for contentName
		if !exists {
			log.Fatalf("(handling %v) relay does not exist for content '%v'\n", remote, contentName)
		}

		// getting sdp file for the stream
		sdp := r.SdpDatabase.MustGetSdp(contentName)
		log.Printf("(handling %v) retrieved sdp file for content '%v'\n", remote, contentName)

		// address to add to the relay
		nextAddress := utils.ReplacePortFromAddressString(remote, relay.Port)

		reply(packets.Port(requestId, contentName, nextAddress, sdp), conn) // reply with streaming port
		log.Printf("(handling %v) responded with 'PORT' packet, addr=%v", remote, nextAddress)

		err := relay.Add(nextAddress) // add client to relay
		utils.Check(err)
		log.Printf("(handling %v) added address '%v' to the relay for '%v'\n", remote, nextAddress, contentName)
		return
	}

	log.Printf("(handling %v) no relay found for content '%v', asking server\n", remote, contentName)
	incoming.Header.Hops++

	// the best server to ask the stream from
	svr := r.GetBestServer(contentName)
	log.Printf("(handling %v) selected server at '%v' for the streaming of '%v'\n", remote, svr.Address, contentName)

	// stopping metrics measurement to avoid conflicts
	svr.TickerChan <- true

	// clearing out the 'pong' dangling packets
	b := make([]byte, 1024)
	_, err := svr.Conn.Read(b)

	utils.Check(err)
	log.Printf("(metrics %v) temporarily stopped metric analysis with server\n", svr.Address)

	defer func() {
		r.measure(svr.Conn)
		log.Printf("(metrics %v) restored metric analysis with server\n", svr.Address)
	}() // resuming metrics after the talk with the server

	// create request packet for the received content name
	packet := packets.Request(contentName)
	buffer, err := packets.Encode[string](packet)
	utils.Check(err)

	// send request packet to the best server
	_, err = svr.Conn.Write(buffer)
	if err != nil {
		log.Printf("(servers %v) cannot write packet 'REQ'\n", svr)
		return
	}
	log.Printf("(servers %v) sent packet 'REQ' for '%v'\n", svr.Address, contentName)

	// receive and decode the response
	responseBuffer := make([]byte, 1024)
	n, err := svr.Conn.Read(responseBuffer)
	if err != nil {
		log.Printf("(servers %v) cannot read packet\n", svr.Address)
		return
	}

	resp, err := packets.Decode[string](responseBuffer[:n])
	utils.Check(err)

	if !resp.Header.Flag.OnlyHasFlag(packets.CSND) {
		log.Printf("(servers %v) did not receive port for stream of '%v'\n", conn.RemoteAddr().String(), contentName)
		reply(
			packets.Miss(requestId, contentName),
			conn,
		)
		log.Printf("(handling %v) sent packet 'MISS', reason 'no port from server'\n", remote)
		return
	}

	log.Printf("(servers %v) received packet 'CSND' with addr=%v\n", svr.Address, resp.Payload)
	log.Printf("(servers %v) server is streaming '%v' at address '%v'\n", svr.Address, contentName, resp.Payload)

	// create new relay
	relayPort := strconv.FormatUint(r.NextPort(), 10)
	relay := node.NewRelay(contentName, resp.Payload, relayPort)
	log.Printf("(handling %v) created new relay for '%v', relay port is '%v'", remote, contentName, relayPort)

	// add the address of the prev node to the relay
	nextAddress := utils.ReplacePortFromAddressString(remote, relay.Port)
	err = relay.Add(nextAddress)
	utils.Check(err)
	log.Printf("(handling %v) added address '%v' to relay for '%v'\n", remote, nextAddress, contentName)

	// add relay to pool
	err = r.AddRelay(contentName, relay)
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

	// getting the sdp file from the server
	sdpReqBytes := make([]byte, 35600)
	n, err = svr.Conn.Read(sdpReqBytes)
	utils.Check(err)

	// decoding and setting sdp file
	sdpPacket, err := packets.Decode[[]byte](sdpReqBytes[:n])
	if sdpPacket.Header.Flag.OnlyHasFlag(packets.SDP) {
		log.Printf("(servers %v) received packet 'SDP' for content '%v'\n", svr.Address, contentName)
		r.SdpDatabase.SetSdp(contentName, sdpPacket.Payload)
		log.Printf("(servers %v) set sdp file for content '%v'\n", svr.Address, contentName)
	} else {
		log.Printf("(servers %v) did not receive sdp file for stream of '%v'\n", remote, contentName)
		reply(
			packets.Miss(requestId, contentName),
			conn,
		)
		log.Printf("(handling %v) sent packet 'MISS', reason 'no sdp from server'\n", remote)
		return
	}

	reply(packets.Port(requestId, contentName, nextAddress, sdpPacket.Payload), conn) // reply to client in which port I'm streaming
	log.Printf("(handling %v) sent packet 'PORT', addr=%v\n", remote, nextAddress)
}

func reply(response packets.Packet, conn net.Conn) {

	enc, err := response.Encode()
	utils.Check(err)

	_, err = conn.Write(enc)
	if err != nil {
		log.Printf("(handling %v) could not write, reason 'unknown'\n", conn.RemoteAddr().String())
	}

}
