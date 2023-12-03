package node

import (
	"errors"
	"github.com/gweebg/mcast/internal/utils"
	"log"
	"net"
	"sync"
)

type Relay struct {
	// Name of the video that we are listening to from Origin.
	ContentName string
	// Address of which the video stream is coming from.
	Origin string

	// RWMutex to handle concurrency when adding new addresses.
	mu sync.RWMutex

	// UDP listener on Origin.
	receiver *net.UDPConn
	// Slice containing the addresses (as *UDPAddr) to forward the bytes to.
	Addresses []*net.UDPAddr // no longer needed as for Connections
	// Default port for forwarding addresses.
	Port string

	Connections []*net.UDPConn
}

// NewRelay creates a new Relay object.
func NewRelay(contentName string, origin string, port string) *Relay {

	addr, err := net.ResolveUDPAddr("udp", origin)
	utils.Check(err)

	conn, err := net.ListenUDP("udp", addr)
	utils.Check(err)

	return &Relay{
		ContentName: contentName,
		Addresses:   make([]*net.UDPAddr, 0),
		Connections: make([]*net.UDPConn, 0),
		Origin:      origin,
		receiver:    conn,
		Port:        port,
	}
}

// Stop stops the reading from Origin by closing the connection.
func (r *Relay) Stop() error {
	log.Printf("stopping relay of '%v' with origin at '%v'\n", r.ContentName, r.Origin)
	return r.receiver.Close()
}

// Add adds a new address into the Relay, this makes so that the bytes read
// from Loop are forwarder to address as well.
func (r *Relay) Add(address string) error {

	r.mu.Lock()
	defer r.mu.Unlock()

	for _, addr := range r.Addresses {
		if addr.String() == address {
			return errors.New("already streaming for address " + address)
		}
	}

	asUdp, err := net.ResolveUDPAddr("udp", address)
	utils.Check(err)

	//udpConn, err := net.DialUDP("udp", nil, asUdp)
	//utils.Check(err)

	r.Addresses = append(r.Addresses, asUdp)
	//r.Connections = append(r.Connections, udpConn)

	return nil
}

// Loop reads a UDP stream from Origin and forwards it to the addresses specified in Addresses.
func (r *Relay) Loop() {

	buffer := make([]byte, 35600)
	for {

		n, _, err := r.receiver.ReadFromUDP(buffer)
		//log.Printf("reading from %v\n", r.Origin)
		if err != nil {
			continue
		}

		r.mu.RLock()

		if len(r.Addresses) > 0 {
			for _, addr := range r.Addresses {
				//_, err := conn.Write(buffer[:n])
				//log.Printf("sent to %v\n", conn.RemoteAddr().String())
				_, err := r.receiver.WriteToUDP(buffer[:n], addr)
				if err != nil {
					// log.Printf("cannot relay packets to %v\n", addr.String())
					continue
				}
			}
		}

		r.mu.RUnlock()
	}
}
