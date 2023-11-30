package rendezvous

import (
	"log"
	"net"
	"net/netip"
	"sync"

	"github.com/gweebg/mcast/internal/handlers"
	"github.com/gweebg/mcast/internal/node"
	"github.com/gweebg/mcast/internal/packets"
	"github.com/gweebg/mcast/internal/utils"
)

// Falta: 
//      comunicacao inicial com os servidores
//      relay quando chega um pedido de streaming 

// ouve tcp:
//   recebe um pedido de discovery -> responde yes
//   recebe um stream request ->
//      verifica se esta a difundir esse conteudo
//          yes: responde com a porta da stream
//          no: comunicar com o server que tiver o conteudo e comecar a stream
//              e responde com a porta
//          para ambos espera que o nodo respoda com ok antes de abrir o udp

type Servers map[string]*ServerInfo

// falta aqui o mutex
type ServerInfo struct {
	Address    string
	PacketLoss uint64
	Jitter     float32
	Latency    float32
	Content    []string
}

func NewServers(addrs []string) Servers {
	srvs := make(Servers)
	for _, a := range addrs {
		srvs[a].Address = a
	}
	return srvs
}

type Rendezvous struct {
	// map key server addr e value struct com struct de metricas conteudo associado
	// ao server e.g. {Fonte:S1; Métrica: 1; Conteúdos: [movie1.mp4, video4.ogg], Estado: ativa }
	Address netip.AddrPort
	Servers Servers

	Requests *node.RequestDb

	// relay pool, keeps track of receiving streams and who are we relaying them to
	RelayPool map[string]*node.Relay
	rMu       sync.RWMutex
    CurrentPort uint16
    
	TCPHandler handlers.TCPConn
}

func New(addrString string, servers ...string) *Rendezvous {
	handler := handlers.NewTCP(
		handlers.WithListenTCP(),
		handlers.WithHandleTCP(Handler),
	)

	addr, err := netip.ParseAddrPort(addrString)
	utils.Check(err)
	return &Rendezvous{
		Address:    addr,
		Servers:    NewServers(servers),
		Requests:   node.NewRequestDb(),
		TCPHandler: *handler,
		RelayPool:  make(map[string]*node.Relay),
	}
}

// Run starts the main listening loop and passes each connection to Handler.
func (r *Rendezvous) Run() {

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
            rendezvous.OnStream(p,conn)
		}

	}

}
// IsStreaming checks whether the current node is streaming a certain content
// by its contentName.
func (r *Rendezvous) IsStreaming(contentName string) bool {
	r.rMu.RLock()
	defer r.rMu.RUnlock()

	if _, exists := r.RelayPool[contentName]; exists {
		return true
	}
	return false
}

// ContentExists checks if a provided content is available in any of the active
// servers.
func (r *Rendezvous) ContentExists(contentName string) bool {
    for k, s := range r.Servers {
        for _, c := range s.Content{
            if c == contentName {
                log.Printf("Found %v in server %v",contentName, k)
                return true
            }
        }
    }
    log.Printf("%v is not available in any of the active servers.",contentName) 
    return false
}



