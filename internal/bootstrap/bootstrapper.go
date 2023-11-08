package bootstrap

import (
	"errors"
	"github.com/gweebg/mcast/internal/utils"
	"log"
	"net"
	"net/netip"
)

type Bootstrap struct {
	Config  Config // no need for mutex since its readonly
	Address string
}

func New(filename, address string) *Bootstrap {
	return &Bootstrap{
		Config:  MustParse(filename),
		Address: address,
	}
}

func (b *Bootstrap) GetNode(addr netip.Addr) (Node, error) {

	if node, exists := b.Config.NodeGroup[addr]; exists {
		return node, nil
	}

	// gob cannot encode nil values, so we set the default for a Node
	return Node{}, errors.New("no records found for " + addr.String())
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

	addr := utils.MustNormalizeAddr(conn.RemoteAddr()) // conn.RemoteAddr as a netip.Addr

	r, err := b.GetNode(addr)
	if err != nil {
		log.Printf("(%v) %v\n", rAddr, err)
	}

	ok := answer(r, rflag, conn)
	if !ok {
		log.Fatalf("error while encoding or sending the response")
	}
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

func answer(node Node, flag FlagType, conn net.Conn) bool {

	resp := Packet{
		Header: PacketHeader{
			Flag:  flag,
			Count: uint(len(node.Neighbours)),
		},
		Payload: node,
	} // response packet

	encResp, err := resp.Encode()
	if err != nil {
		log.Print("cannot encode response packet")
		return false
	}

	s, err := conn.Write(encResp)
	if err != nil {
		log.Print("cannot write to socket")
		return false
	}

	log.Printf("(%v) responded with %v bytes\n", conn.RemoteAddr().String(), s)
	return true
}
