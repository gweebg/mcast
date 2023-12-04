package node

import (
	"github.com/gweebg/mcast/internal/server"
	"github.com/gweebg/mcast/internal/streamer"
	"log"
	"net"
	"net/netip"
	"strings"

	"github.com/gweebg/mcast/internal/bootstrap"
	"github.com/gweebg/mcast/internal/handlers"
	"github.com/gweebg/mcast/internal/node/databases"
	"github.com/gweebg/mcast/internal/packets"
	"github.com/gweebg/mcast/internal/utils"
)

type Node struct {

	// Listener - tcp listener at 0.0.0.0:<port_by_bootstrap>
	Listener handlers.TCPConn

	// HandledRequests - contains which requests were already handled by this node
	HandledRequests *databases.RequestDb

	// RelayPool - group of the relays streaming content of this node
	RelayPool *databases.RelayDb

	// PositiveRequests - database of positive answers received
	PositiveRequests *databases.PositiveDb

	// Servers - database of connected servers in case of rendezvous
	Servers *databases.Servers

	// Self - specific information about this node
	Self bootstrap.Node

	// Flooder - floods neighbours with a message, only accounting for the first positive one
	Flooder streamer.Flooder
}

func New(bootstrapper string) *Node {

	self, err := setupSelf(bootstrapper)
	utils.Check(err)

	listener := handlers.NewTCP(
		handlers.WithListenTCP(),
		handlers.WithHandleTCP(Handler),
	)

	node := &Node{
		Self:            self,
		HandledRequests: databases.NewRequestDb(),
		RelayPool:       databases.NewRelayDb(9000),
		Listener:        *listener,
	}

	switch self.Type {

	case bootstrap.ONode:

		log.Printf("node is an overlay node\n")
		node.PositiveRequests = databases.NewPositiveDb()
		node.Flooder = streamer.NewFlooder(self.Neighbours)
		return node

	case bootstrap.RendezvousPoint:

		log.Printf("node is a rendezvous node\n")
		serverAddresses := make([]string, 0)

		for _, srv := range self.Neighbours {
			serverAddresses = append(serverAddresses, srv.String())
		}

		node.Servers = databases.NewServers(serverAddresses)

		for _, srv := range self.Neighbours {
			node.connectToServer(srv.String())
		}

		return node

	default:
		log.Fatalf("node must be either of type 'node' | 'rendezvous', but got '%v'\n", self.Type)
		return nil
	}

}

// Run starts the main listening loop and passes each connection to Handler.
func (n *Node) Run() {

	listenString := "0.0.0.0:" + strings.Split(n.Self.SelfIp, ":")[1]

	listenAddress, err := netip.ParseAddrPort(listenString)
	utils.Check(err)

	n.Listener.Listen(
		listenAddress,
		n.Listener.Handle,
		n,
	)
}

// Handler reads from the connection conn and distributes the packets
// through the available handlers at handler.go
func Handler(conn net.Conn, va ...interface{}) {

	node := va[0].(*Node) // get the current node

	addrString := conn.RemoteAddr().String()
	log.Printf("(handling %v) new client connected\n", addrString)

	// read the connection for incoming data.
	buffer := make([]byte, ReadSize)
	for {

		// read message from the client
		n, err := conn.Read(buffer)
		if err != nil {
			continue
		}

		// decode the packet
		p, err := packets.DecodePacket(buffer[:n])
		if err != nil {
			log.Printf("(handling %v) malformed packet, ignoring...\n", addrString)
			utils.CloseConnection(conn, addrString)
			return
		}

		if node.Self.Type == bootstrap.ONode {

			switch p.Header.Flags {

			case packets.DISC:
				node.NodeOnDiscovery(p, conn)

			case packets.STREAM:
				node.NodeOnStream(p, conn)

			}

		} else {

			switch p.Header.Flags {

			case packets.DISC:
				node.RendOnDiscovery(p, conn)

			case packets.STREAM:
				node.RendOnStream(p, conn)

			}
		}
	}
}

func (n *Node) connectToServer(servAddr string) {

	tcpAddr, err := net.ResolveTCPAddr("tcp", servAddr)
	utils.Check(err)

	// setup tcp connection with server
	conn, err := net.DialTCP("tcp", nil, tcpAddr)
	utils.Check(err)

	// send wake packet to server
	wakePacket := packets.Wake()
	p, err := packets.Encode[string](wakePacket)
	utils.Check(err)

	_, err = conn.Write(p)
	utils.Check(err)

	// wait for response
	buffer := make([]byte, ReadSize)
	s, err := conn.Read(buffer)
	utils.Check(err)

	recv, err := packets.Decode[[]server.ConfigItem](buffer[:s])
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
	serverInfo, exists := n.Servers.GetServer(servAddr)
	if !exists {
		log.Fatalf("server '%v' does not exist in server database, but it should\n", servAddr)
	}

	serverInfo.Content = formatted
	serverInfo.Conn = conn

	serverInfo.Measure()
}
