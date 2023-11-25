package node

import (
	"github.com/gweebg/mcast/internal/packets"
	"github.com/gweebg/mcast/internal/utils"
	"log"
	"net"
	"net/netip"
	"sync"
)

type Flooder struct {
	Neighbours []netip.AddrPort
}

func NewFlooder(n []netip.AddrPort) Flooder {
	return Flooder{Neighbours: n}
}

// Flood sends a specialized message to each neighbour only taking
// into account the first answer to arrive, ignoring the others.
func (f Flooder) Flood(packet packets.Packet) (packets.Packet, bool) {

	var wg sync.WaitGroup

	response := make(chan packets.Packet)
	done := make(chan struct{})

	for _, neighbour := range f.Neighbours {
		wg.Add(1)
		go f.sendTo(neighbour, packet, &wg, response, done)
	}

	select {
	case res := <-response:
		return res, true // Return the first response received
	case <-done:
		return packets.Packet{}, false
	}

}

// sendTo, sends a packet (content) to the specified neighbour (dest)
// once a response is received, signals the response channel and closes the done channel
// indicating to other goroutines that a response was already received.
func (f Flooder) sendTo(dest netip.AddrPort, content packets.Packet, wg *sync.WaitGroup, response chan packets.Packet, done chan struct{}) {

	defer wg.Done()

	// connection setup
	conn, err := net.Dial("tcp", dest.String()) // todo: check this value
	if err != nil {
		log.Printf("cannot connect to '%v' via tcp\n%v", dest.String(), err.Error())
		return
	}
	defer utils.CloseConnection(conn, dest.String())

	// packet encoding
	buffer, err := content.Encode()
	utils.Check(err)

	// sending packet to dest
	_, err = conn.Write(buffer)
	if err != nil {
		log.Printf("cannot write packet to '%v'\n%v", dest.String(), err.Error())
		return
	}

	// reading and decoding the response
	responseBuffer := make([]byte, 1024)
	n, err := conn.Read(responseBuffer)
	if err != nil {
		log.Printf("cannot read packet from '%v'\n%v", dest.String(), err.Error())
		return
	}

	resp, err := packets.DecodePacket(responseBuffer[:n])
	utils.Check(err)

	// updating response state
	select {

	case response <- resp:
		close(done)

	default:
		// we do nothing
	}
}
