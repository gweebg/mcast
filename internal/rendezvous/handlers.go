package rendezvous

import (
	"log"
	"net"
	"strconv"

	"github.com/gweebg/mcast/internal/node"
	"github.com/gweebg/mcast/internal/packets"
	"github.com/gweebg/mcast/internal/server"
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

	requestId := incoming.Header.RequestId
	contentName := incoming.Payload.ContentName

	defer func(r *Rendezvous) {
		r.Requests.Set(requestId, true)
	}(r) // set packet as handled

	if r.Requests.IsHandled(requestId) {
		reply(
			packets.Miss(requestId, contentName),
			conn,
		)
		return
	} // packet was already handled

	if r.ContentExists(contentName) { // this content is available

		log.Printf("content '%v' is available for streaming\n", contentName)
		// todo: check value of r.Address.String()
		reply(
			packets.Found(requestId, contentName, r.Address.String()),
			conn,
		)
		return

	} else { // this content is not available in any server

		log.Printf("content '%v' is not available for streaming\n", contentName)
		reply(
			packets.Miss(requestId, contentName),
			conn,
		)
	}
}

func (r *Rendezvous) OnStream(incoming packets.Packet, conn net.Conn) {
	/*
		am I streaming the content ?
			yes -> reply with port and add address to corresponding relay
			no  -> select best server for the content, create and init relay, reply with port
	*/

	requestId := incoming.Header.RequestId
	contentName := incoming.Payload.ContentName

	log.Printf("(%v) received stream packet from '%v'\n", requestId, incoming.Header.Source)
	defer utils.CloseConnection(conn, conn.RemoteAddr().String())

	if r.IsStreaming(contentName) { // if am I streaming contentName

		relay, exists := r.RelayPool[contentName] // get correct relay for contentName
		if !exists {
			log.Fatalf("relay does not exist for content '%v'\n", contentName)
		}

		// address to add to the relay
		address := utils.ReplacePortFromAddressString(conn.RemoteAddr().String(), relay.Port)

		reply(packets.Port(requestId, contentName, relay.Port), conn) // reply with streaming port

		err := relay.Add(address) // add client to relay
		utils.Check(err)

		return
	}

	incoming.Header.Hops++

	svr := r.GetBestServerConn(contentName)

	// create request packet for the received content name
	packet := packets.Request(contentName)
	buffer, err := packets.Encode[string](packet)

	utils.Check(err)

	// send request packet to the best server
	_, err = svr.Write(buffer)
	log.Printf("sent request for '%v' to server '%v'\n", contentName, svr.RemoteAddr().String())
	if err != nil {
		log.Printf("(handlers.go) cannot write packet to '%v'\n", svr)
		return
	}

	// receive and decode the response
	responseBuffer := make([]byte, 1024)
	n, err := svr.Read(responseBuffer)
	if err != nil {
		log.Printf("(handlers.go) cannot read packet from '%v'\n", svr.RemoteAddr().String())
		return
	}

	resp, err := packets.Decode[int](responseBuffer[:n])
	utils.Check(err)

	if !resp.Header.Flag.OnlyHasFlag(server.CSND) {
		log.Printf("asked server '%v' for '%v', but recieved nothing\n", conn.RemoteAddr().String(), contentName)
		reply(
			packets.Miss(requestId, contentName),
			conn,
		)
		return
	}

	videoSourcePort := strconv.FormatInt(int64(resp.Payload), 10)                        // the port on which the server is streaming
	videoSource := utils.ReplacePortFromAddressString("127.0.0.1:0000", videoSourcePort) // the full streaming address

	log.Printf("server is streaming '%v' at address '%v'\n", contentName, videoSource)

	// create new relay
	relayPort := strconv.FormatUint(r.NextPort(), 10)
	relay := node.NewRelay(contentName, videoSource, relayPort)

	// add relay to pool
	err = r.AddRelay(contentName, relay)
	utils.Check(err)

	go relay.Loop() // start the relay streaming loop

	reply(packets.Port(requestId, contentName, relay.Port), conn) // reply to client in which port I'm streaming
}

func reply(response packets.Packet, conn net.Conn) {

	enc, err := response.Encode()
	utils.Check(err)

	_, err = conn.Write(enc)
	if err != nil {
		log.Printf("(handlers.go) could not write to '%v'\n", conn.RemoteAddr().String())
	}

}
