package server

import (
	"errors"
	"github.com/gweebg/mcast/internal/flags"
	"github.com/gweebg/mcast/internal/handlers"
	"github.com/gweebg/mcast/internal/packets"
	"github.com/gweebg/mcast/internal/utils"
	"log"
	"net"
	"net/netip"
	"strconv"
	"sync"
)

type Packet struct {
	packets.BasePacket[string]
}

const (
	WAKE flags.FlagType = 0b1
	CONT flags.FlagType = 0b10
	CSND flags.FlagType = 0b100
	STOP flags.FlagType = 0b1000
)

type Server struct {
	Address netip.AddrPort
	Config  Config

	TCPHandler handlers.TCPConn

	Connections map[string]*Connection // todo: convert to its own struct
	AccessPort  uint64
	mu          sync.Mutex
}

func NewServer(addr, path string) *Server {

	tcpHandler := handlers.NewTCP(
		handlers.WithListenTCP(),
		handlers.WithHandleTCP(Handler),
	) // tcp handler for this server

	addrPort, err := netip.ParseAddrPort(addr)
	utils.Check(err) // address string to AddrPort obj

	return &Server{
		Address:     addrPort,
		Config:      utils.MustParseJson[Config](path, ValidateConfig),
		TCPHandler:  *tcpHandler,
		Connections: make(map[string]*Connection),
		AccessPort:  20000,
	} // server instantiation

}

func (s *Server) Run() {
	s.TCPHandler.Listen(
		s.Address,           // remote address
		s.TCPHandler.Handle, // request handler
		s,                   // arguments passed to the handler function
	)
}

func (s *Server) AddConnection(conn net.Conn) (*Connection, error) {

	s.mu.Lock()
	defer s.mu.Unlock()

	_, exists := s.Connections[conn.RemoteAddr().String()]
	if exists {
		return nil, errors.New("connection already exists")
	}

	c := NewConnection(conn)
	s.Connections[conn.RemoteAddr().String()] = c

	return c, nil
}

func (s *Server) RemoveConnection(address string) error {

	s.mu.Lock()
	defer s.mu.Unlock()

	c, exists := s.Connections[address] // get the connection
	if !exists {
		return errors.New("cannot remove non existent connection " + address)
	}

	c.Shutdown() // close every tcp/udp connection

	delete(s.Connections, address) // delete from connection pool
	log.Printf("(%v) connection removed from pool", address)

	return nil
}

func (s *Server) GetConnection(address string) (*Connection, error) {

	conn, exists := s.Connections[address]
	if exists {
		return conn, nil
	}

	return nil, errors.New("somehow the connection does not exist (this is very bad)")

}

func (s *Server) GetPort() uint64 {

	port := s.AccessPort
	s.AccessPort++

	return port
}

func Handler(conn net.Conn, va ...interface{}) {

	s := va[0].(*Server) // get the server

	addrString := conn.RemoteAddr().String()
	log.Printf("(%v) client connected\n", addrString)

	// read the connection for incoming data.
	buffer := make([]byte, 1024)
	for { // todo: wont timeout due to the incoming metric packets

		_, err := conn.Read(buffer)
		if err != nil {
			utils.CloseConnection(conn, addrString)
			break
		}

		p, err := packets.Decode[string](buffer) // decode packet
		if err != nil {
			log.Printf("(%v) malformed packet, ignoring...\n", addrString)
			continue // just ignore the packet, continue the read
		}

		c, err := s.GetConnection(addrString) // get connection (streamers management)
		if err != nil {
			c, err = s.AddConnection(conn) // if this is the first message, then we add it to the pool
			utils.Check(err)
		}

		// handle packet todo: is this really necessary ????
		packet := Packet{p}

		switch packet.Header.Flag {

		case WAKE: // received WAKE sends CONT
			s.HandleWakeCall(c)

		}
	}
}

func (s *Server) HandleWakeCall(c *Connection) {

	remote := c.Conn.RemoteAddr().String()

	// response packet
	pac := ContentInfoPacket(s.Config.Content)

	encPac, err := packets.Encode[[]ConfigItem](pac) // encode the packet
	utils.Check(err)

	size, err := c.Conn.Write(encPac) // send the packet
	if err != nil {
		log.Print(err.Error())

		err = s.RemoveConnection(remote)
		utils.Check(err) // panic if the connection is already gone
	}

	log.Printf("(%v) sent CONT packet (%d)\n", remote, size)
}

func (s *Server) HandleContent(p Packet, c *Connection) {

	remote := c.Conn.RemoteAddr().String()

	// create response packet
	pack := ContentAccessPacket(s.GetPort())
	encPack, err := packets.Encode[uint64](pack)
	utils.Check(err)

	// send response packet
	size, err := c.Conn.Write(encPack)
	if err != nil {
		log.Print(err.Error())

		err = s.RemoveConnection(remote)
		utils.Check(err)
	}
	log.Printf("(%v) sent CONT packet (%d)\n", remote, size)

	// get the address where to stream the video
	portString := strconv.FormatUint(s.GetPort(), 10)
	streamAddr := s.Address.Addr().String() + portString // todo: check this value

	// create and initialize the udp connection
	streamer, err := c.CreateStreamer(streamAddr, p.Payload) // todo: fail
	utils.Check(err)

	// start the video streaming
	go streamer.Start()
}

func HandleStop(p Packet) (Packet, error) {
	// update connection pool, stop content stream
	return Packet{}, nil
}
