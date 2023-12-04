package bootstrap

import (
	"errors"
	"log"
	"strings"

	"github.com/gweebg/mcast/internal/packets"
	"github.com/gweebg/mcast/internal/utils"
	"net"
)

type Bootstrap struct {
	// Config - configuration file containing the structure of the topology.
	Config Config

	// ListenAddress - address of this node, also obtained from the configuration file.
	ListenAddress string
}

func New(configFilename string) *Bootstrap {
	b := &Bootstrap{
		Config: utils.MustParseJson[Config](configFilename, ValidateConfig),
	}

	b.ListenAddress = "0.0.0.0:" + strings.Split(b.Config.Bootstrapper.SelfIp, ":")[1]
	return b
}

func (b *Bootstrap) Listen() {

	l, err := net.Listen("tcp", b.ListenAddress)
	utils.Check(err)

	defer func() {
		err := l.Close()
		utils.Check(err)
	}()

	log.Printf("(listener %v) bootstrapper listening...\n", b.ListenAddress)

	for {
		conn, err := l.Accept()
		utils.Check(err)
		go b.handle(conn)
	}
}

func (b *Bootstrap) handle(conn net.Conn) {

	remote := conn.RemoteAddr().String()
	readSize := 1024
	rflag := SEND // response flag

	log.Printf("(handling %v) new client connected\n", remote)

	defer func() {
		err := conn.Close()
		utils.Check(err)
	}()

	// read the connection for incoming data
	buffer := make([]byte, readSize)

	size, err := conn.Read(buffer)
	utils.Check(err)

	// decode the incoming packet
	_, err = decodeAndCheckPacket(buffer[:size]) // p, holds the data from the client.
	if err != nil {
		log.Printf("(handling %v) malformed packet, %v\n", remote, err.Error())

		err = answer(Node{}, ERR, conn)
		if err != nil {
			log.Fatalf("(handling %v) %v\n", remote, err.Error())
		}

		return
	}

	// getting the node
	node, err := b.GetNode(remote)
	if err != nil {
		log.Printf("(handling %v) %v\n", remote, err)

		err = answer(Node{}, ERR, conn)
		if err != nil {
			log.Fatalf("(handling %v) %v\n", remote, err.Error())
		}

		return
	}

	// reply with the fetched node to the client
	err = answer(node, rflag, conn)
	if err != nil {
		log.Fatalf("(handling %v) %v\n", remote, err.Error())
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

func answer(node Node, flag utils.FlagType, conn net.Conn) error {

	resp := packets.BasePacket[Node]{
		Header: packets.PacketHeader{
			Flag: flag,
		},
		Payload: node,
	} // response packet

	encResp, err := packets.Encode[Node](resp)
	if err != nil {
		return errors.New("cannot encode response packet")
	}

	s, err := conn.Write(encResp)
	if err != nil {
		return errors.New("cannot write to socket")
	}

	log.Printf("(handling %v) responded with %v bytes\n", conn.RemoteAddr().String(), s)
	return nil
}
