package node

import (
	"log"
	"net"

	"github.com/gweebg/mcast/internal/bootstrap"
	"github.com/gweebg/mcast/internal/packets"
	"github.com/gweebg/mcast/internal/utils"
)

type Node struct {
	bootstrapAddr string
}

func New(bootstrapAddr string) *Node {
	return &Node{
		bootstrapAddr,
	}
}

func (n *Node) setupBootstrapper() {

	tcpAddr, err := net.ResolveTCPAddr("tcp", n.bootstrapAddr)
	if err != nil {
		log.Fatalf(err.Error())
	}

	sndPacket := packets.BasePacket[string]{
		Header: packets.PacketHeader{
			Flag: bootstrap.GET,
		},
		Payload: "",
	}

	p, err := packets.Encode[string](sndPacket)
	if err != nil {
		log.Fatal(err.Error())
	}
	conn, err := net.DialTCP("tcp", nil, tcpAddr)
	if err != nil {
		log.Fatalf(err.Error())
	}

	_, err = conn.Write(p)
	if err != nil {
		log.Fatalf(err.Error())
	}

	buffer := make([]byte, 1024)
	_, err = conn.Read(buffer)
	if err != nil {
		log.Fatalf(err.Error())
	}

	recv, err := packets.Decode[bootstrap.Node](buffer)
	if err != nil {
		log.Fatal(err.Error())
	}
	if recv.Header.Flag.OnlyHasFlag(bootstrap.SEND) {
		utils.PrintStruct(recv)
	} else {
		log.Fatalf("Recieved %v instead of SEND", recv.Header.Flag)
	}
}

func (n *Node) Run() {
	// base de dados de conexoes
	// comunicacar com o bootstrapper para receber os seus vizinhos
	n.setupBootstrapper()
}
