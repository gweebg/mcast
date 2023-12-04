package bootstrap

import (
	"errors"
	"log"
	"strings"

	"github.com/gweebg/mcast/internal/packets"
	"github.com/gweebg/mcast/internal/utils"
	"net"
)

type Packet = packets.BasePacket[Node]

const (
	SEND utils.FlagType = 0b1
	ERR  utils.FlagType = 0b10
	GET  utils.FlagType = 0b100
)

type Bootstrap struct {
	Config  Config // no need for mutex since its readonly
	Address string
}

func New(filename, address string) *Bootstrap {
	return &Bootstrap{
		Config:  utils.MustParseJson[Config](filename, ValidateConfig),
		Address: address,
	}
}

func (b *Bootstrap) GetNode(addrPort string) (Node, error) {

	addr := strings.Split(addrPort, ":")[0]

	if node, exists := b.Config.NodeGroup[addr]; exists {
		return node, nil
	}

	// gob cannot encode nil values, so we set the default for a Node
	return Node{}, errors.New("no records found for " + addr)
}

func (b *Bootstrap) Listen() {

	l, err := net.Listen("tcp", b.Address)
	if err != nil {
		log.Fatal(err.Error())
	}

	defer func(l net.Listener) {
		err := l.Close()
		if err != nil {
			log.Fatal("could not close the connection.")
		}
	}(l) // Closing the connection once the function ends.

	log.Println("listening...")
	for {
		conn, err := l.Accept()
		if err != nil {
			log.Fatal(err.Error())
		}

		go b.handle(conn)
	} // Infinite loop that attends incoming clients.
}

func (b *Bootstrap) handle(conn net.Conn) {

	rAddr := conn.RemoteAddr().String()
	rflag := SEND // response flag

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
	size, err := conn.Read(buffer)
	if err != nil {
		log.Fatalf("could not read from %v\n", rAddr)
	}

	_, err = decodeAndCheckPacket(buffer[:size]) // p, holds the data from the client.
	if err != nil {
		log.Printf("(%v) %v\n", rAddr, err)
		rflag = ERR
	}

	// todo: skip db check if rflag already ERR
	addr := conn.RemoteAddr().String() // conn.RemoteAddr as a netip.Addr

	r, err := b.GetNode(addr)
	if err != nil {
		log.Printf("(%v) %v\n", rAddr, err)
		rflag = ERR
	}

	ok := answer(r, rflag, conn)
	if !ok {
		log.Fatalf("error while encoding or sending the response")
	}
}

func decodeAndCheckPacket(data []byte) (p packets.BasePacket[string], err error) {

	p, err = packets.Decode[string](data) // p, holds the data from the client.
	if err != nil {
		err = errors.New("could not decode packet, skipping")
	} // Trying to decode the packet from the client.

	if !p.Header.Flag.OnlyHasFlag(GET) {
		err = errors.New("no GET flag on packet header, skipping")
	} // Does the packet 'make sense?'

	return p, err
}

func answer(node Node, flag utils.FlagType, conn net.Conn) bool {

	resp := Packet{
		Header: packets.PacketHeader{
			Flag: flag,
		},
		Payload: node,
	} // response packet

	encResp, err := packets.Encode[Node](packets.BasePacket[Node](resp))
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
