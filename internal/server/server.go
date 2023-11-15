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
	Address    netip.AddrPort
	Config     Config
	TCPHandler handlers.TCPConn

	Connections map[string]*Connection
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
	} // server instantiation

}

func (s *Server) Run() {
	s.TCPHandler.Listen(
		s.Address,           // remote address
		s.TCPHandler.Handle, // request handler
		s,                   // arguments passed to the handler function // todo: wtf ?
	)
}

func (s *Server) AddConnection(conn net.Conn) error {

	_, exists := s.Connections[conn.RemoteAddr().String()]
	if exists {
		return errors.New("connection already exists")
	}

	s.Connections[conn.RemoteAddr().String()] = NewConnection(conn)
	return nil

}

func Handler(conn net.Conn, va ...interface{}) {

	s := va[0].(*Server) // get the server

	addrString := conn.RemoteAddr().String()
	log.Printf("(%v) client connected\n", addrString)

	//defer utils.CloseConnection(conn, addrString), we don't want to close the connection right away

	// read the connection for incoming data.
	buffer := make([]byte, 1024) // todo: max packet size ?

	_, err := conn.Read(buffer)
	if err != nil {
		log.Fatalf("could not read from %v\n", addrString)
	}

	p, err := packets.Decode[string](buffer) // decode packet
	if err != nil {
		log.Printf("(%v) malformed packet, ignoring...\n", addrString)
		utils.CloseConnection(conn, addrString)
		return
	}

	s.mu.Lock()
	err = s.AddConnection(conn) // update connection pool
	if err != nil {
		utils.CloseConnection(conn, addrString)
		log.Fatal(err.Error())
	}
	s.mu.Unlock()

	// handle packet
	packet := Packet{p} // convert packet to structured format

	switch packet.Header.Flag {

	case WAKE: // send CONT, send the config file

	}
}

func HandleWakeCall(p Packet) (Packet, error) {
	// send the config file, keep the connection open
	return Packet{}, nil
}

func HandleContent(p Packet) (Packet, error) {
	// create new StreamingConn, update connection pool, stream content over udp
	return Packet{}, nil
}

func HandleStop(p Packet) (Packet, error) {
	// update connection pool, stop content stream
	return Packet{}, nil
}
