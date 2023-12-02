package server

import (
	"github.com/gweebg/mcast/internal/flags"
	"github.com/gweebg/mcast/internal/handlers"
	"github.com/gweebg/mcast/internal/packets"
	"github.com/gweebg/mcast/internal/streamer"
	"github.com/gweebg/mcast/internal/utils"
	"log"
	"net"
	"net/netip"
	"strconv"
)

type Packet struct {
	packets.BasePacket[string]
}

const (
	WAKE flags.FlagType = 0b1
    CONT flags.FlagType = 0b10
	CSND flags.FlagType = 0b100
	STOP flags.FlagType = 0b1000
	OK   flags.FlagType = 0b10000
	REQ  flags.FlagType = 0b100000
)

type Server struct {
	Address netip.AddrPort
	Config  Config

	TCPHandler handlers.TCPConn

	ConnectionPool streamer.StreamingPool
	AccessPort     int
}

// New creates a new server instance when passed its operating address
// and its configuration file path.
func New(addr, path string) *Server {

	tcpHandler := handlers.NewTCP(
		handlers.WithListenTCP(),
		handlers.WithHandleTCP(Handler),
	) // tcp handler for this server

	addrPort, err := netip.ParseAddrPort(addr)
	utils.Check(err) // address string to AddrPort obj

	return &Server{
		Address:        addrPort,
		Config:         utils.MustParseJson[Config](path, ValidateConfig),
		TCPHandler:     *tcpHandler,
		ConnectionPool: streamer.NewStreamingPool(),
		AccessPort:     20000,
	} // server instantiation
}

// Run function is responsible for running the main loop of the server.
func (s *Server) Run() {
	s.TCPHandler.Listen(
		s.Address,           // remote address
		s.TCPHandler.Handle, // request handler
		s,                   // arguments passed to the handler function
	)
}

// Handler is the function that decides what On* event handler to call.
func Handler(conn net.Conn, va ...interface{}) {

	s := va[0].(*Server) // get the server

	addrString := conn.RemoteAddr().String()
	log.Printf("(%v) client connected\n", addrString)

	// read the connection for incoming data.
	buffer := make([]byte, 1024)
	for { // todo: wont timeout due to the incoming metric packets

		n, err := conn.Read(buffer) // read from connection
		if err != nil {
			continue // keep the connection open
		}

		p, err := packets.Decode[string](buffer[:n]) // decode packet
		if err != nil {
			log.Printf("(%v) malformed packet, ignoring...\n", addrString)
			continue // just ignore the packet, continue the read
		}

		switch p.Header.Flag {

		case WAKE: // received WAKE
			s.OnWake(conn)

		case REQ: // received REQ
			s.OnContent(conn, p)

		case STOP: // received STOP
			s.OnStop(conn, p)

		}
	}
}

// OnWake handles the request 'WAKE' from the client (rendezvous point).
// Once this kind of request arrives, the server will answer with a list
// of ConfigItem representing what content it can stream.
func (s *Server) OnWake(conn net.Conn) {

	remote := conn.RemoteAddr().String()
	log.Printf("(%v) received packet with header 'WAKE'\n", remote)

	// response packet
	pac := ContentInfoPacket(s.Config.Content)

	encPac, err := packets.Encode[[]ConfigItem](pac) // encode the packet
	utils.Check(err)

	size, err := conn.Write(encPac) // send the packet
	utils.Check(err)

	log.Printf("(%v) answered with packet 'CSND' (%d bytes)\n", remote, size)
}

// OnContent handles the request 'REQ' from the client.
// First the server responds via TCP (conn net.Conn) with the port where the content
// will be streamed on. Once the client answers with an 'OK' packet then we start the
// UDP stream by utilizing our streamer.Streamer struct.
// todo: refactor this function into to abstract certain parts
func (s *Server) OnContent(conn net.Conn, p packets.BasePacket[string]) {

	remote := conn.RemoteAddr().String()
	log.Printf("(%v) received packet with header 'REQ'\n", remote)

	// create response packet
	port := s.AccessPort
	streamers, err := s.ConnectionPool.GetPool(remote)
	if err == nil { // if there's already a streaming pool the streaming port is the default + len(pool)
		port += len(streamers) + 1
	}

	encPack, err := packets.Encode[int](ContentPortPacket(port))
	utils.Check(err)

	// send response packet
	size, err := conn.Write(encPack)
	utils.Check(err)

	log.Printf("(%v) answered with packet 'CONT' (%d bytes)\n", remote, size)

	// get the address where to stream the video
	portString := strconv.FormatInt(int64(port), 10)
	streamAddr := s.Address.Addr().String() + ":" + portString
	log.Printf("(%v) setting up streaming at '%v'\n", remote, streamAddr)

	response := make([]byte, 1024) // receiving the clients response
	n, err := conn.Read(response)
	utils.Check(err)

	recvPack, err := packets.Decode[string](response[:n])
	utils.Check(err)

	if recvPack.Header.Flag.OnlyHasFlag(OK) {

		log.Printf("(%v) received response with header 'OK'\n", remote)

		// create and initialize the streamer object responsible for the content streaming
		stmr := streamer.New(
			streamer.WithAddress(streamAddr),
			streamer.WithContentName(p.Payload),
		)

		err = s.ConnectionPool.Add(remote, stmr) // create a new streaming pool
		utils.Check(err)

		go stmr.Stream()
		log.Printf("(%v) started streaming %v on %v\n", remote, p.Payload, streamAddr)

	} else {
		log.Printf("(%v) unexpected response, was expecting packet with header 'OK'", remote)
	}

}

// OnStop function is responsible for stopping the transmission of a certain content.
// With every connection (alongside its streaming connections) being stored in a StreamingPool
// we can assure that we can stop its streaming goroutines.
func (s *Server) OnStop(conn net.Conn, p packets.BasePacket[string]) {

	remote := conn.RemoteAddr().String()
	log.Printf("(%v) received packet with header 'STOP'\n", remote)

	err := s.ConnectionPool.Delete(remote, p.Payload) // delete already handles the streamer teardown
	utils.Check(err)

	log.Printf("(%v) stopped streaming %v\n", remote, p.Payload)
}
