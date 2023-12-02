package rendezvous

import (
	"errors"
	"log"
	"net"
	"net/netip"
	"sync"
	"time"

	"github.com/gweebg/mcast/internal/handlers"
	"github.com/gweebg/mcast/internal/node"
	"github.com/gweebg/mcast/internal/packets"
	"github.com/gweebg/mcast/internal/server"
	"github.com/gweebg/mcast/internal/utils"
)

type Servers map[string]*ServerInfo

type ServerInfo struct {

	// the address where the node will talk to the server.
	Address string

	// packet loss of the server.
	PacketLoss uint64
	// Jitter of the server.
	Jitter float32
	// Latency of the server.
	Latency float32
	// Metric lock to prevent race conditions.
	mMu sync.Mutex

	// which content the server has available.
	Content []server.ConfigItem
	// tcp connection to the server at Address.
	Conn *net.TCPConn

	Ticker     *time.Ticker
	tickerChan chan bool
}

// CalculateMetrics calculates the general metrics of a server, the Latency is given
// a weight of 60% while the Jitter is given a weight of 40%. Packet loss is not accounted
// for the metrics calculation, yet.
func (s *ServerInfo) CalculateMetrics() float32 {
	return ((s.Latency * 0.6) + (s.Jitter * 0.4)) / 100
}

// NewServers populate the Servers struct with the ServerInfo objects by passing
// their addresses.
func NewServers(addrs []string) Servers {
	srvs := make(Servers)

	for _, addr := range addrs {
		srvs[addr] = &ServerInfo{
			Address:    addr,
			Content:    make([]server.ConfigItem, 0),
			tickerChan: make(chan bool),
		}
	}

	return srvs
}

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
		CurrentPort: 20000,
		RelayPool:   make(map[string]*node.Relay),
	}
}

// Run starts the main listening loop and passes each connection to Handler.
func (r *Rendezvous) Run() {

	for _, srv := range r.Servers {
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
	log.Printf("(%v) client connected\n", addrString)

	// read the connection for incoming data.
	buffer := make([]byte, 1024)
	for {

		n, err := conn.Read(buffer) // read from connection
		if err != nil {
			log.Printf("(%v) could not read from connection, closing conn\n", addrString)
			utils.CloseConnection(conn, addrString)
			return
		}

		p, err := packets.DecodePacket(buffer[:n]) // decode packet
		if err != nil {
			log.Printf("(%v) malformed packet, ignoring...\n", addrString)
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

	log.Printf("received '%v' from '%v', conext wake request\n", recv.Payload, servAddr)

	// setting content and connection for the server
	r.sMu.Lock()
	if _, exists := r.Servers[servAddr]; !exists {
		r.Servers[servAddr].Content = recv.Payload
		r.Servers[servAddr].Conn = conn
	}
	r.sMu.Unlock()

	r.measure(conn) // start metrics measuring process
}

func (r *Rendezvous) measure(conn *net.TCPConn) {

	srv, exists := r.Servers[conn.RemoteAddr().String()]
	if !exists {
		log.Fatalf("server '%v' should exists, but it doesn't\n", conn.RemoteAddr().String())
	}

	srv.Ticker = time.NewTicker(5 * time.Second)

	go func() {
		for {
			select {

			case <-srv.tickerChan:
				return

			case <-srv.Ticker.C:

				ping := packets.Ping()

				packet, err := packets.Encode[string](ping)
				utils.Check(err)

				startTime := time.Now()

				_, err = conn.Write(packet)
				utils.Check(err)

				_, err = conn.Read(nil)
				utils.Check(err)

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

	log.Printf("measuring metrics for server '%v'\n", srv.Address)
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

// GetBestServerConn returns the connection to the best server (better metric)
// with the content (contentName) available.
func (r *Rendezvous) GetBestServerConn(contentName string) *net.TCPConn {
	var best *ServerInfo

	for _, srv := range r.Servers {

		if !utils.Contains(srv.Content, contentName) { // if server does not contentName, skip iteration
			continue
		}

		if best == nil || best.CalculateMetrics() < srv.CalculateMetrics() {
			best = srv
		}
	}
	return best.Conn
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
