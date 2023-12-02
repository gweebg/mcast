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
		log.Printf("(handling %v) send 'FOUND' with source address as '%v'\n", r.Address.String())
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

		// address to add to the relay

		reply(packets.Port(requestId, contentName, relay.Port), conn) // reply with streaming port
		log.Printf("(handling %v) responded with 'PORT' packet, port=%v", remote, relay.Port)

		address := utils.ReplacePortFromAddressString(conn.RemoteAddr().String(), relay.Port)
		err := relay.Add(address) // add client to relay
		utils.Check(err)

		log.Printf("(handling %v) added address '%v' to the relay for '%v'\n", remote, address, contentName)
		return
	}

	log.Printf("(handling %v) no relay found for content '%v', asking server\n", remote, contentName)
	incoming.Header.Hops++
	svr := r.GetBestServerConn(contentName)

	log.Printf("(handling %v) selected server at '%v' for the streaming of '%v'\n", remote, svr.RemoteAddr().String(), contentName)

	// create request packet for the received content name
	packet := packets.Request(contentName)
	buffer, err := packets.Encode[string](packet)

	utils.Check(err)

	// send request packet to the best server
	_, err = svr.Write(buffer)
	if err != nil {
		log.Printf("(servers %v) cannot write packet 'REQ'\n", svr)
		return
	}
	log.Printf("(servers %v) sent packet 'REQ' for '%v'\n", svr.RemoteAddr().String(), contentName)

	// receive and decode the response
	responseBuffer := make([]byte, 1024)
	n, err := svr.Read(responseBuffer)
	if err != nil {
		log.Printf("(servers %v) cannot read packet\n", svr.RemoteAddr().String())
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

	log.Printf("(servers %v) received packet 'CSND' with port=%v\n", svr.RemoteAddr().String(), resp.Payload)

	videoSource := utils.ReplacePortFromAddressString("127.0.0.1:9999", resp.Payload) // the full streaming address
	log.Printf("(servers %v) server is streaming '%v' at address '%v'\n", svr.RemoteAddr().String(), contentName, videoSource)

	// create new relay
	relayPort := strconv.FormatUint(r.NextPort(), 10)
	relay := node.NewRelay(contentName, videoSource, relayPort)
	log.Printf("(handling %v) created new relay for '%v', relay port is '%v'", remote, contentName, relayPort)

	// add relay to pool
	err = r.AddRelay(contentName, relay)
	utils.Check(err)
	log.Printf("(handling %v) added relay for '%v' to the pool\n", remote, contentName)

	go relay.Loop() // start the relay streaming loop
	log.Printf("(handling %v) relay started transmitting '%v' with origin at '%v'\n", remote, contentName, videoSource)

	ok := packets.Ok()
	okResponse, err := packets.Encode[string](ok)
	utils.Check(err)

	_, err = svr.Write(okResponse)
	if err != nil {
		log.Fatalf("(servers %v) cannot reply with 'OK' to server\n", svr.RemoteAddr().String())
	}
	log.Printf("(servers %v) sent packet 'OK'\n", svr.RemoteAddr().String())

	reply(packets.Port(requestId, contentName, relay.Port), conn) // reply to client in which port I'm streaming
	log.Printf("(handling %v) sent packet 'PORT', port=%v\n", remote, relay.Port)
}

func reply(response packets.Packet, conn net.Conn) {

	enc, err := response.Encode()
	utils.Check(err)

	_, err = conn.Write(enc)
	if err != nil {
		log.Printf("(handling %v) could not write, reason 'unknown'\n", conn.RemoteAddr().String())
	}

}
