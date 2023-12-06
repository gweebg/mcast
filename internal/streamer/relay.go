package streamer

import (
	"errors"
	"github.com/gweebg/mcast/internal/utils"
	"log"
	"net"
	"strings"
	"sync"
)

type Relay struct {

	// ContentName - name of the video that we are listening to from Origin.
	ContentName string

	// Origin - address of which the video stream is coming from.
	Origin string

	// mu - to handle concurrency when adding new addresses.
	mu sync.RWMutex

	// receiver - udp listener on Origin.
	receiver *net.UDPConn

	// Addresses - slice containing the addresses (as *UDPAddr) to forward the bytes to.
	Addresses []*net.UDPAddr

	// Port - default port for forwarding addresses.
	Port string

	// StopChannel - channel that is signaled when we want to stop the Loop() routine.
	StopChannel chan struct{}
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
		Origin:      origin,
		receiver:    conn,
		Port:        port,
		StopChannel: make(chan struct{}),
	}
}

// Stop stops the reading from Origin by closing the connection.
func (r *Relay) Stop() error {
	log.Printf("stopping relay of '%v' with origin at '%v'\n", r.ContentName, r.Origin)
	r.StopChannel <- struct{}{}
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

	r.Addresses = append(r.Addresses, asUdp)

	return nil
}

func (r *Relay) Remove(address string) {

	r.mu.Lock()
	defer r.mu.Unlock()

	filtered := make([]*net.UDPAddr, 0)
	for _, addr := range r.Addresses {

		if strings.Contains(address, ":") {
			address = strings.Split(address, ":")[0]
		}

		if addr.IP.String() != address {
			filtered = append(filtered, addr)
		} else {
			log.Printf("did not add address %v (correct is %v)\n", addr.IP.String(), address)
		}
	}

	r.Addresses = filtered
}

// Loop reads a UDP stream from Origin and forwards it to the addresses specified in Addresses.
func (r *Relay) Loop() {

	buffer := make([]byte, TsMtu*10)
	for {

		select {

		case <-r.StopChannel:
			return

		default:

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
}
