package rendezvous

import (
	"errors"
	"log"
	"net"
	"net/netip"
	"strings"
	"sync"
	"time"

	"github.com/gweebg/mcast/internal/handlers"
	"github.com/gweebg/mcast/internal/node"
	"github.com/gweebg/mcast/internal/packets"
	"github.com/gweebg/mcast/internal/server"
	"github.com/gweebg/mcast/internal/utils"
)

type Rendezvous struct {
	// Address of the Rendezvous node, cannot be localhost or 127.0.0.1.
	Address netip.AddrPort

	// currently known servers.
	Servers Servers
	// Servers mutex, to prevent race conditions.
	sMu sync.RWMutex

	// keeps track of handled requests.
	Requests *node.RequestDb

	// keeps track of receiving streams and who are we relaying them to.
	RelayPool map[string]*node.Relay
	// relay pool mutex, to prevent race conditions.
	rMu sync.RWMutex
	// current operating port when creating new relays.
	CurrentPort uint64

	// tcp listener for incoming requests from other network nodes.
	TCPHandler handlers.TCPConn
}

// New creates a new Rendezvous node, returns a pointer.
func New(addrString string, servers ...string) *Rendezvous {

	handler := handlers.NewTCP(
		handlers.WithListenTCP(),
		handlers.WithHandleTCP(Handler),
	) // tcp listener & handler

	addr, err := netip.ParseAddrPort(addrString) // self address, cannot be localhost/127.0.0.1
	utils.Check(err)

	return &Rendezvous{
		Address:     addr,
		Servers:     NewServers(servers),
		Requests:    node.NewRequestDb(),
		TCPHandler:  *handler,
		CurrentPort: 9000,
		RelayPool:   make(map[string]*node.Relay),
	}
}

// Run starts the main listening loop and passes each connection to Handler.
func (r *Rendezvous) Run() {

	for _, srv := range r.Servers {
		log.Printf("(setup) retrieving information from server '%v'\n", srv.Address)
		r.connectToServer(srv.Address)
	}

	r.TCPHandler.Listen(
		r.Address,
		r.TCPHandler.Handle,
		r,
	)
}

// Handler reads from the connection conn and distributes the packets
// through the available handlers at handler.go
func Handler(conn net.Conn, va ...interface{}) {

	rendezvous := va[0].(*Rendezvous) // get rendezvous

	addrString := conn.RemoteAddr().String()
	log.Printf("(handling %v) new client connected\n", addrString)

	// read the connection for incoming data.
	buffer := make([]byte, 1024)
	for {

		n, err := conn.Read(buffer) // read from connection
		if err != nil {
			continue
			//log.Printf("(read %v) could not read from connection\n", addrString)
			//utils.CloseConnection(conn, addrString)
			//return
		}

		p, err := packets.DecodePacket(buffer[:n]) // decode packet
		if err != nil {
			log.Printf("(decode %v) malformed packet, ignoring...\n", addrString)
			utils.CloseConnection(conn, addrString)
			return
		}

		switch p.Header.Flags {

		case packets.DISC:
			rendezvous.OnDiscovery(p, conn)

		case packets.STREAM:
			rendezvous.OnStream(p, conn)

		}
	}
}

func (r *Rendezvous) connectToServer(servAddr string) {

	// setup tcp connection with server
	tcpAddr, err := net.ResolveTCPAddr("tcp", servAddr)
	utils.Check(err)

	conn, err := net.DialTCP("tcp", nil, tcpAddr)
	utils.Check(err)

	// send wake packet to server
	wakePacket := packets.Wake()
	p, err := packets.Encode[string](wakePacket)
	utils.Check(err)

	_, err = conn.Write(p)
	utils.Check(err)

	// wait for response
	buffer := make([]byte, 1024)
	n, err := conn.Read(buffer)
	utils.Check(err)

	recv, err := packets.Decode[[]server.ConfigItem](buffer[:n])
	utils.Check(err)

	// removing the full path from the content names
	formatted := make([]server.ConfigItem, 0)
	for _, config := range recv.Payload {

		nameList := strings.Split(config.Name, "/")
		name := nameList[len(nameList)-1]

		config.Name = name

		formatted = append(formatted, config)
	}

	log.Printf("(server %v) received server information:\n", servAddr)
	utils.PrintStruct(formatted)

	// setting content and connection for the server
	r.sMu.Lock()
	r.Servers[servAddr].Content = formatted
	r.Servers[servAddr].Conn = conn
	r.sMu.Unlock()

	r.measure(conn) // start metrics measuring process
}

func (r *Rendezvous) measure(conn *net.TCPConn) {

	remote := conn.RemoteAddr().String()

	srv, exists := r.Servers[conn.RemoteAddr().String()]
	if !exists {
		log.Fatalf("(server %v) server should exist in Servers, but it doesn't\n", remote)
	}

	srv.Ticker = time.NewTicker(5 * time.Second)

	go func() {
		log.Printf("(metrics %v) started metrics loop\n", remote)
		for {
			select {

			case <-srv.TickerChan:
				return

			case <-srv.Ticker.C:

				ping := packets.Ping()

				packet, err := packets.Encode[string](ping)
				utils.Check(err)

				startTime := time.Now()

				_, err = conn.Write(packet)
				utils.Check(err)

				log.Printf("(metrics %v) sent ping\n", remote)

				_, err = conn.Read(nil)
				utils.Check(err)

				log.Printf("(metrics %v) got pong\n", remote)

				stopTime := time.Now()

				elapsedTime := stopTime.Sub(startTime).Seconds()

				srv.mMu.Lock()
				srv.Latency = float32(elapsedTime)
				srv.Jitter = float32(elapsedTime)
				srv.mMu.Unlock()

				// todo: metric setting
			}
		}
	}()

}

// ContentExists checks if a provided content is available in any of the active
// servers.
func (r *Rendezvous) ContentExists(contentName string) bool {

	for _, srv := range r.Servers {
		for _, content := range srv.Content {
			if content.Name == contentName {
				return true
			}
		}
	}

	return false
}

// GetBestServer returns the connection to the best server (better metric)
// with the content (contentName) available.
func (r *Rendezvous) GetBestServer(contentName string) *ServerInfo {
	var best *ServerInfo

	for _, srv := range r.Servers {

		if !Contains(srv.Content, contentName) { // if server does not contentName, skip iteration
			continue
		}

		if best == nil || best.CalculateMetrics() < srv.CalculateMetrics() {
			best = srv
		}
	}
	return best
}

// IsStreaming checks whether the current node is streaming a certain content by its contentName.
func (r *Rendezvous) IsStreaming(contentName string) bool {
	r.rMu.RLock()
	defer r.rMu.RUnlock()

	if _, exists := r.RelayPool[contentName]; exists {
		return true
	}
	return false
}

func (r *Rendezvous) NextPort() uint64 {
	port := r.CurrentPort
	r.CurrentPort++
	return port
}

func (r *Rendezvous) AddRelay(contentName string, relay *node.Relay) error {

	r.rMu.Lock()
	defer r.rMu.Unlock()

	_, exists := r.RelayPool[contentName]
	if exists {
		return errors.New("relay for content '" + contentName + "' already exists.")
	}

	r.RelayPool[contentName] = relay
	return nil
}
