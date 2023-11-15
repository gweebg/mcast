package server

import (
	"github.com/gweebg/mcast/internal/flags"
	"github.com/gweebg/mcast/internal/handlers"
	"github.com/gweebg/mcast/internal/packets"
	"github.com/gweebg/mcast/internal/utils"
	"log"
	"net"
	"net/netip"
)

type Packet struct {
	packets.BasePacket[string]
}

const (
	WAKE flags.FlagType = 0b1
	CONT flags.FlagType = 0b10
	CSND flags.FlagType = 0b100
	STOP flags.FlagType = 0b1000
) // todo: when does the connection get closed ?

type Server struct {
	Address    netip.AddrPort
	Config     Config
	TCPHandler handlers.TCPConn

	Connections map[string]*Connection
}

func NewServer(addr, path string) *Server {

	tcp := handlers.NewTCP(
		handlers.WithListenTCP(),
		handlers.WithHandleTCP(Handler),
	) // tcp handler for this server

	addrPort, err := netip.ParseAddrPort(addr)
	if err != nil {
		log.Fatal(err.Error())
	} // address string to AddrPort obj

	return &Server{
		addrPort,
		utils.MustParseJson[Config](path, ValidateConfig),
		*tcp,
		make(map[string]*Connection),
	} // server instantiation

}

func (s *Server) Run() {
	s.TCPHandler.Listen(s.Address, s.TCPHandler.Handle)
}

func Handler(conn net.Conn, va ...interface{}) {

	addrString := conn.RemoteAddr().String()
	log.Printf("(%v) client connected\n", addrString)

	defer utils.CloseConnection(conn, addrString)

	// read the connection for incoming data.
	buffer := make([]byte, 1024) // todo: max packet size ?

	_, err := conn.Read(buffer)
	if err != nil {
		log.Fatalf("could not read from %v\n", addrString)
	}

	p, err := packets.Decode[string](buffer)
	if err != nil {
		log.Printf("(%v)malformed packet, ignoring...\n", addrString)
		return
	}

	packet := Packet{p}
	switch packet.Header.Flag {

	case WAKE: // send CONT, send the config file
		//todo

	}
}

func HandleWakeCall(p Packet) (Packet, error) {

	return Packet{}, nil
}

func HandleContent(p Packet) (Packet, error) {
	return Packet{}, nil
}

func HandleStop(p Packet) (Packet, error) {
	return Packet{}, nil
}
