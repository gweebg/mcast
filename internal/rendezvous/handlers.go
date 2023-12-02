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

func reply(response packets.Packet, conn net.Conn) {

	enc, err := response.Encode()
	utils.Check(err)

	_, err = conn.Write(enc)
	if err != nil {
		log.Printf("(handlers.go) could not write to '%v'\n", conn.RemoteAddr().String())
	}

}

func (r *Rendezvous) OnDiscovery(incoming packets.Packet, conn net.Conn) {
	/*
				did I handle the packet ? (check requestId in db)
				  no ->  do i have this content?
					    yes -> answer with FOUND
		                no -> answer with a MISS
				  yes -> ignore
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

	if r.ContentExists(contentName) {
		// this content is available
		// todo: verificar valor do r.Address.String()
		reply(
			packets.Found(requestId, contentName, r.Address.String()),
			conn,
		)
		return
	} else {
		reply(
			packets.Miss(requestId, contentName),
			conn,
		)
	}
}

func (r *Rendezvous) OnStream(incoming packets.Packet, conn net.Conn) {
	/*
		am I streaming the content ?
			yes -> reply with PORT

			no -> did I receive a positive answer for this request id before ?
				yes -> forward packet to that address & wait for PORT
				no -> reply with NOEXISTS
	*/

	requestId := incoming.Header.RequestId
	contentName := incoming.Payload.ContentName
	log.Printf("(%v) received stream packet from '%v'\n", requestId, incoming.Header.Source)

	defer utils.CloseConnection(conn, conn.RemoteAddr().String())

	if r.IsStreaming(contentName) {

		relay, exists := r.RelayPool[contentName] // get relay for contentName
		if !exists {
			log.Fatalf("relay does not exist for content '%v'\n", contentName)
		}

		address := utils.ReplacePortFromAddressString(conn.RemoteAddr().String(), relay.Port)

		reply(packets.Port(requestId, contentName, relay.Port), conn) // send port

		err := relay.Add(address) // add client to relay
		utils.Check(err)

		return
	}

	incoming.Header.Hops++

	// todo: get content from best server, create and add stream to relay
	bestServ := r.GetBestServer(contentName)

	packet := packets.Request(contentName)
	// enviar CONT a source
	buffer, err := packets.Encode[string](packet)
	utils.Check(err)

	// sending packet to dest
	_, err = bestServ.Write(buffer)
	if err != nil {
		log.Printf("(handlers.go) cannot write packet to '%v'\n", bestServ)
		return
	}

	// receber CSND
	// reading and decoding the response
	responseBuffer := make([]byte, 1024)
	n, err := bestServ.Read(responseBuffer)
	if err != nil {
		log.Printf("(handlers.go) cannot read packet from '%v'\n", bestServ.RemoteAddr().String())
		return
	}

	resp, err := packets.Decode[int](responseBuffer[:n])
	utils.Check(err)

	if !resp.Header.Flag.OnlyHasFlag(server.CSND) {
		log.Printf("Asked server %v for %v, but recieved nothing\n", conn.RemoteAddr().String(), contentName)
		reply(
			packets.Miss(requestId, contentName),
			conn,
		)
		return
	}

    videoSourcePort := strconv.FormatInt(int64(resp.Payload),10)
	videoSource := utils.ReplacePortFromAddressString("127.0.0.1:0000", videoSourcePort)

	relayPort := strconv.FormatUint(r.NextPort(), 10)
	relay := node.NewRelay(contentName, videoSource, relayPort)

	err = r.AddRelay(contentName, relay)
	utils.Check(err)

	go relay.Loop()

	reply(packets.Port(requestId, contentName, relay.Port), conn)
	return
}
