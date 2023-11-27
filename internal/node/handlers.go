package node

import (
	"github.com/gweebg/mcast/internal/packets"
	"github.com/gweebg/mcast/internal/utils"
	"log"
	"net"
	"net/netip"
)

func reply(response packets.Packet, conn net.Conn) {

	enc, err := response.Encode()
	utils.Check(err)

	_, err = conn.Write(enc)
	if err != nil {
		log.Printf("(handlers.go) could not write to '%v'\n", conn.RemoteAddr().String())
	}

}

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

	defer func(n *Node) {
		n.Requests.Set(requestId, true)
	}(n) // set packet as handled

	if n.Requests.IsHandled(requestId) {
		reply(
			packets.Miss(requestId, contentName),
			conn,
		)
		return
	} // packet was already handled

	if n.IsStreaming(contentName) {
		reply(
			packets.Found(requestId, contentName, n.Self.SelfIp),
			conn,
		)
		return
	} // i'm already streaming the content

	addrPort, err := netip.ParseAddrPort(conn.RemoteAddr().String())
	utils.Check(err)

	if len(filterNeighbour(addrPort, n.Self.Neighbours)) > 0 {

		incoming.Header.Hops++

		// response, holds the answer resulted from the flooding
		response, _ := n.Flooder.Flood(incoming, addrPort)

		if response.Header.Flags.OnlyHasFlag(packets.FOUND) {
			n.SetPositive(requestId, response.Header.Source)
		}

		reply(response, conn)
	}
}
