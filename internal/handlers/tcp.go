package handlers

import (
	"log"
	"net"
	"net/netip"
)

type TCPHandle func(net.Conn, ...interface{})
type TCPListen func(netip.AddrPort, TCPHandle)

type TCPConn struct {
	Handle TCPHandle
	Listen TCPListen
}

func NewTCP(options ...func(conn *TCPConn)) *TCPConn {

	conn := &TCPConn{}
	for _, o := range options {
		o(conn)
	}
	return conn

}

func WithListenTCP(l ...TCPListen) func(*TCPConn) {
	switch len(l) {

	case 0:
		return func(s *TCPConn) {
			s.Listen = defaultTCPListen
		}

	default:
		return func(s *TCPConn) {
			s.Listen = l[0]
		}
	}
}

func WithHandleTCP(h ...TCPHandle) func(*TCPConn) {
	switch len(h) {

	case 0:
		return func(s *TCPConn) {
			s.Handle = defaultTCPHandle
		}

	default:
		return func(s *TCPConn) {
			s.Handle = h[0]
		}
	}
}

func defaultTCPListen(addr netip.AddrPort, handle TCPHandle) {

	l, err := net.Listen("tcp", addr.String())
	if err != nil {
		log.Fatal(err.Error())
	}

	defer func(l net.Listener) {
		err := l.Close()
		if err != nil {
			log.Fatalf("could not close the connection at %v\n", addr)
		}
	}(l) // closing the connection once the function ends.

	log.Printf("(%v) server is listening...\n", addr)
	for {
		conn, err := l.Accept()
		if err != nil {
			log.Fatal(err.Error())
		}

		go handle(conn) // handle the request
	}

}

func defaultTCPHandle(conn net.Conn, va ...interface{}) {

	addrString := conn.RemoteAddr().String()

	log.Printf("(%v) new client connected\n", addrString)

	defer func(conn net.Conn) {
		err := conn.Close()
		if err != nil {
			log.Fatalf("could not close the connection at %v\n", addrString)
		}
		log.Printf("(%v) closed connection\n", addrString)
	}(conn) // defer the closing of the connection.

	// read the connection for incoming data.
	buffer := make([]byte, 1024)
	_, err := conn.Read(buffer)
	if err != nil {
		log.Fatalf("could not read from %v\n", addrString)
	}

	// do nothing with the read data since this is pretty much dummy function

}
