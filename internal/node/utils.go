package node

import (
	"errors"
	"github.com/gweebg/mcast/internal/bootstrap"
	"github.com/gweebg/mcast/internal/packets"
	"github.com/gweebg/mcast/internal/utils"
	"log"
	"net"
)

const ReadSize = 1024

var (
	Request = packets.BasePacket[string]{
		Header: packets.PacketHeader{
			Flag: bootstrap.GET,
		},
		Payload: "",
	}
)

func setupSelf(bootstrapAddr string) (bootstrap.Node, error) {

	tcpAddr, err := net.ResolveTCPAddr("tcp", bootstrapAddr)
	utils.Check(err)

	conn, err := net.DialTCP("tcp", nil, tcpAddr)
	utils.Check(err)

	p, err := packets.Encode[string](Request)
	utils.Check(err)

	_, err = conn.Write(p)
	utils.Check(err)

	buffer := make([]byte, ReadSize)
	size, err := conn.Read(buffer)
	utils.Check(err)

	response, err := packets.Decode[bootstrap.Node](buffer[:size])
	utils.Check(err)

	if response.Header.Flag.OnlyHasFlag(bootstrap.SEND) {
		return response.Payload, nil
	}

	return bootstrap.Node{}, errors.New("expected flag SEND from bootstrapper, but received another")
}

func Reply(response packets.Packet, conn net.Conn) {

	enc, err := response.Encode()
	utils.Check(err)

	_, err = conn.Write(enc)
	if err != nil {
		log.Printf("(handlers.go) could not write to '%v'\n", conn.RemoteAddr().String())
	}

}

func Follow(packet packets.Packet, destination string) (packets.Packet, error) {

	// connection setup
	conn, err := net.Dial("tcp", destination)
	if err != nil {
		log.Printf("(handlers.go) cannot connect to '%v' via tcp\n", destination)
		return packets.Packet{}, err
	}
	defer utils.CloseConnection(conn, destination)

	// packet encoding
	buffer, err := packet.Encode()
	utils.Check(err)

	// sending packet to dest
	_, err = conn.Write(buffer)
	if err != nil {
		log.Printf("(handlers.go) cannot write packet to '%v'\n", destination)
		return packets.Packet{}, err
	}

	// reading and decoding the response
	responseBuffer := make([]byte, 1024)
	n, err := conn.Read(responseBuffer)
	if err != nil {
		log.Printf("(handlers.go) cannot read packet from '%v'\n", destination)
		return packets.Packet{}, err
	}

	resp, err := packets.DecodePacket(responseBuffer[:n])
	utils.Check(err)

	return resp, nil
}
