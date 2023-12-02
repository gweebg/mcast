package server

import (
	"github.com/gweebg/mcast/internal/handlers"
	"github.com/gweebg/mcast/internal/packets"
	"github.com/gweebg/mcast/internal/streamer"
	"github.com/gweebg/mcast/internal/utils"

	"log"
	"net"
	"net/netip"
	"strconv"
)

type Server struct {
	// Address in which the server listen for tcp connections.
	Address netip.AddrPort
	// represents the server properties and what content it has available.
	Config Config

	// tcp listener on the Address
	TCPHandler handlers.TCPConn

	// contains the addresses where I'm streaming to and what content is being streamed
	ConnectionPool streamer.StreamingPool
	// default streamer port, incremented depending on the number of streamers
	AccessPort int
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
		AccessPort:     8000,
	} // server instantiation
}

// Run function is responsible for running the main loop of the server.
func (s *Server) Run() {

	utils.PrintStruct(s)

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
	log.Printf("(handling %v) new connection received\n", addrString)

	// read the connection for incoming data.
	buffer := make([]byte, 1024)
	for {

		n, err := conn.Read(buffer) // read from connection
		if err != nil {
			continue // keep the connection open
		}

		p, err := packets.Decode[string](buffer[:n]) // decode packet
		if err != nil {
			log.Printf("(handling %v) malformed packet, ignoring...\n", addrString)
			continue // just ignore the packet, continue the read
		}

		switch p.Header.Flag {

		case packets.WAKE: // received WAKE
			s.OnWake(conn)

		case packets.REQ: // received REQ
			s.OnContent(conn, p)

		case packets.STOP: // received STOP
			s.OnStop(conn, p)

		case packets.PING: // received PING
			s.OnPing(conn)

		}
	}
}

// OnWake handles the request 'WAKE' from the client (rendezvous point).
// Once this kind of request arrives, the server will answer with a list
// of ConfigItem representing what content it can stream.
func (s *Server) OnWake(conn net.Conn) {

	remote := conn.RemoteAddr().String()
	log.Printf("(handling %v) received packet with header 'WAKE'\n", remote)

	// response packet
	pac := ContentInfoPacket(s.Config.Content)

	encPac, err := packets.Encode[[]ConfigItem](pac) // encode the packet
	utils.Check(err)

	size, err := conn.Write(encPac) // send the packet
	utils.Check(err)

	log.Printf("(handling %v) answered with packet 'CSND' (%d bytes)\n", remote, size)
}

// OnContent handles the request 'REQ' from the client.
// First the server responds via TCP (conn net.Conn) with the port where the content
// will be streamed on. Once the client answers with an 'OK' packet then we start the
// UDP stream by utilizing our streamer.Streamer struct.
func (s *Server) OnContent(conn net.Conn, p packets.BasePacket[string]) {

	remote := conn.RemoteAddr().String()
	log.Printf("(handling %v) received packet with header 'REQ'\n", remote)

	// create response packet
	port := s.AccessPort // port that we're sending to the client

	streamers, err := s.ConnectionPool.GetPool(remote)
	if err == nil { // if there's already a streaming pool the streaming port is the default + len(pool)
		port += len(streamers) + 1
	}

	// encode the packet with the port as a string
	encPack, err := packets.Encode[string](ContentPortPacket(strconv.FormatInt(int64(port), 10)))
	utils.Check(err)

	// send response packet
	_, err = conn.Write(encPack)
	utils.Check(err)

	log.Printf("(handling %v) answered with packet 'CONT' (port: %d)\n", remote, port)

	// get the address where to stream the video
	portString := strconv.FormatInt(int64(port), 10)
	streamAddr := utils.ReplacePortFromAddressString(conn.RemoteAddr().String(), portString)

	log.Printf("(handling %v) setting up streaming of '%v' at '%v'\n", remote, p.Payload, streamAddr)
	log.Printf("(handling %v) waiting for confirmation...\n", remote)

	response := make([]byte, 1024) // receiving the clients response
	n, err := conn.Read(response)
	utils.Check(err)

	recvPack, err := packets.Decode[string](response[:n])
	utils.Check(err)

	if recvPack.Header.Flag.OnlyHasFlag(packets.OK) {

		log.Printf("(handling %v) received confirmation packet with header 'OK'\n", remote)

		// create and initialize the streamer object responsible for the content streaming
		stmr := streamer.New(
			streamer.WithAddress(streamAddr),
			streamer.WithContentName(p.Payload),
		)
		log.Printf("(handling %v) created new streamer for '%v'\n", remote, p.Payload)

		err = s.ConnectionPool.Add(remote, stmr) // create a new streaming pool
		utils.Check(err)
		log.Printf("(handling %v) added streamer for '%v' to the pool\n", remote, p.Payload)

		go stmr.Stream()

	} else {
		log.Printf("(handling %v) did not receive confirmation, was expecting packet with header 'OK'\n", remote)
	}

}

// OnStop function is responsible for stopping the transmission of a certain content.
// With every connection (alongside its streaming connections) being stored in a StreamingPool
// we can assure that we can stop its streaming goroutines.
func (s *Server) OnStop(conn net.Conn, p packets.BasePacket[string]) {

	remote := conn.RemoteAddr().String()
	log.Printf("(handling %v) received packet with header 'STOP'\n", remote)

	err := s.ConnectionPool.Delete(remote, p.Payload) // delete already handles the streamer teardown
	utils.Check(err)

	log.Printf("(handling %v) stopped streaming %v\n", remote, p.Payload)
}

// OnPing function answers with Pong to Ping requests, used
// in metrics measurements by the clients, such as latency,
// jitter and packet loss.
func (s *Server) OnPing(conn net.Conn) {

	remote := conn.RemoteAddr().String()
	log.Printf("(handling %v) received packet with header 'PING'\n", remote)

	encPack, err := packets.Encode[string](Pong())
	utils.Check(err)

	_, err = conn.Write(encPack)
	if err != nil {
		log.Printf("(handling %v) cannot respond with pong to rendezvous point\n", remote)
	}

	log.Printf("(handling %v) responded with 'Pong'\n", remote)

}
