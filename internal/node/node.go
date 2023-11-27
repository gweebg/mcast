package node

import (
	"github.com/google/uuid"
	"github.com/gweebg/mcast/internal/bootstrap"
	"github.com/gweebg/mcast/internal/handlers"
	"github.com/gweebg/mcast/internal/packets"
	"github.com/gweebg/mcast/internal/utils"
	"log"
	"net"
	"net/netip"
	"sync"
)

type Node struct {

	// information about what type of node I am, what are my neighbours
	Self bootstrap.Node

	// component responsible for flooding its neighbours with a packet
	Flooder Flooder

	// responsible for receiving tcp connections and handling them accordingly
	TCPHandler handlers.TCPConn

	// default address where the client listen to incoming requests
	Address netip.AddrPort

	// keeps track of handled requests
	Requests *RequestDb

	// relay pool, keeps track of receiving streams and who are we relaying them to
	RelayPool map[string]*Relay
	rMu       sync.RWMutex

	// positive, keeps track of received FOUND packets
	Positive map[uuid.UUID]string
	pMu      sync.RWMutex
}

// New creates a new instance of a *Node.
func New(bootstrapAddr string, port string) *Node {

	self, err := setupSelf(bootstrapAddr)
	utils.Check(err)

	handler := handlers.NewTCP(
		handlers.WithListenTCP(),
		handlers.WithHandleTCP(Handler),
	)

	addr, err := netip.ParseAddrPort(self.SelfIp + port)
	utils.Check(err)

	return &Node{
		Self:       self,
		Flooder:    NewFlooder(self.Neighbours),
		TCPHandler: *handler,
		Address:    addr,
		Requests:   NewRequestDb(),
		RelayPool:  make(map[string]*Relay),
	}
}

// Run starts the main listening loop and passes each connection to Handler.
func (n *Node) Run() {

	n.TCPHandler.Listen(
		n.Address,
		n.TCPHandler.Handle,
		n,
	)
}

// Handler reads from the connection conn and distributes the packets
// through the available handlers at handler.go
func Handler(conn net.Conn, va ...interface{}) {

	node := va[0].(*Node) // get the current node

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
			node.OnDiscovery(p, conn)
		}
	}

}

// SetPositive registers that we received a FOUND packet for the request with requestId
// and came from source. Only the first to come is registered.
func (n *Node) SetPositive(requestId uuid.UUID, source string) {
	n.pMu.Lock()
	defer n.pMu.Unlock()

	_, exists := n.Positive[requestId]
	if exists {
		return
	}

	n.Positive[requestId] = source
}

// IsStreaming checks whether the current node is streaming a certain content
// by its contentName.
func (n *Node) IsStreaming(contentName string) bool {
	n.rMu.RLock()
	defer n.rMu.RUnlock()

	if _, exists := n.RelayPool[contentName]; exists {
		return true
	}
	return false
}
