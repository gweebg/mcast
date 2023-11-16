package main

import (
	"log"
	"net"

	"github.com/gweebg/mcast/internal/packets"
	"github.com/gweebg/mcast/internal/server"
	"github.com/gweebg/mcast/internal/utils"
)

func main() {
	servAddr := "127.0.0.1:20010"
	tcpAddr, err := net.ResolveTCPAddr("tcp", servAddr)
	if err != nil {
		log.Fatalf(err.Error())
	}

	packet := packets.BasePacket[string]{
		Header: packets.PacketHeader{
			Flag: server.WAKE,
		},
	}
	p, err := packets.Encode[string](packet)
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

	recv, err := packets.Decode[[]server.ConfigItem](buffer)
	if err != nil {
		log.Fatal(err.Error())
	}
	utils.PrintStruct(recv)
}
