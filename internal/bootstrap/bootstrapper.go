package bootstrap

import (
	"errors"
	"log"
	"net"
	"strings"
)

type NeighbourDatabase map[string][]net.IP

type Bootstrap struct {
	Neighbours NeighbourDatabase // no need for mutex since its readonly
	Address    string
	QuitCh     chan struct{}
}

func New(filename, address string) *Bootstrap {
	return &Bootstrap{
		Neighbours: MustParse(filename),
		Address:    address,
		QuitCh:     make(chan struct{}),
	}
}

func (b *Bootstrap) Listen() {

	l, err := net.Listen("tcp", b.Address)
	if err != nil {
		log.Fatal(err)
	}

	defer func(l net.Listener) {
		err := l.Close()
		if err != nil {
			log.Fatal("Could not close the connection.")
		}
	}(l) // Closing the connection once the function ends.

	log.Println("listening...")
	for {
		conn, err := l.Accept()
		if err != nil {
			log.Fatal(err)
		}

		go b.handle(conn)
	} // Infinite loop that attends incoming clients.
}

func (b *Bootstrap) handle(conn net.Conn) {

	rAddr := conn.RemoteAddr().String()
	rflag := SND // response flag

	log.Printf("(%v) new client connected \n", rAddr)

	defer func(conn net.Conn) {
		err := conn.Close()
		if err != nil {
			log.Fatalf("unable to close connection with %v\n", rAddr)
		}
		log.Printf("(%v) closed connection\n", conn.RemoteAddr().String())
	}(conn) // defer the closing of the connection.

	// read the connection for incoming data.
	buffer := make([]byte, 1024)
	_, err := conn.Read(buffer)
	if err != nil {
		log.Fatalf("could not read from %v\n", rAddr)
	}

	_, err = decodeAndCheckPacket(buffer) // p, holds the data from the client.
	if err != nil {
		log.Printf("(%v) %v\n", rAddr, err)
		rflag = ERR
	}

	r, err := b.GetRecords(conn.RemoteAddr())
	if err != nil {
		log.Printf("(%v) %v\n", rAddr, err)
		r = make([]net.IP, 0) // gob cannot encode nil values, so we set the default for an empty slice
	}

	resp := Packet{
		Header: PacketHeader{
			Flag:  rflag,
			Count: uint(len(r)),
		},
		Payload: r,
	} // response packet

	encResp, err := resp.Encode()
	if err != nil {
		log.Fatal("cannot encode response packet")
	}

	s, err := conn.Write(encResp)
	if err != nil {
		log.Fatal("could not reply to client")
	}
	log.Printf("(%v) responded with %v bytes\n", rAddr, s)

}

func (b *Bootstrap) GetRecords(addr net.Addr) (r []net.IP, err error) {

	parsedAddr := strings.Split(addr.String(), ":")[0]
	r, ok := b.Neighbours[parsedAddr]

	if !ok {
		err = errors.New("no records stored for " + parsedAddr)
	}

	return r, err
}

func decodeAndCheckPacket(data []byte) (p Packet, err error) {

	p, err = Decode(data) // p, holds the data from the client.
	if err != nil {
		err = errors.New("could not decode packet, skipping")
	} // Trying to decode the packet from the client.

	if p.Header.Flag != GET {
		err = errors.New("no GET flag on packet header, skipping")
	} // Does the packet 'make sense?'

	return p, err
}
