package rendezvous

import (
	"log"
	"net"

	"github.com/gweebg/mcast/internal/packets"
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

// todo: adapt to Rendezvous
func (n *Rendezvous) OnStream(incoming packets.Packet, conn net.Conn) {
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

			videoSource := utils.ReplacePortFromAddressString(source, response.Payload.Port)

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
