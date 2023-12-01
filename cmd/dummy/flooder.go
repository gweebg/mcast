package main

import (
	"github.com/google/uuid"
	"github.com/gweebg/mcast/internal/packets"
	"github.com/gweebg/mcast/internal/utils"
	"log"
	"net"
	"os"
	"strconv"
	"time"
)

func main() {

	servAddr := "127.0.0.1:" + os.Args[1]
	log.Println(servAddr)

	list, err := net.Listen("tcp", servAddr)
	utils.Check(err)

	for {
		conn, err := list.Accept()
		utils.Check(err)

		buffer := make([]byte, 1024)
		_, err = conn.Read(buffer)

		log.Printf("received packet\n")

		packet := packets.Discovery(uuid.New(), os.Args[3])

		if yes, _ := strconv.ParseBool(os.Args[2]); !yes {
			packet.Header.Flags.SetFlag(0b10000)
		}

		if sleep, _ := strconv.ParseBool(os.Args[4]); sleep {
			time.Sleep(2 * time.Second)
		}

		utils.PrintStruct(packet)

		p, _ := packet.Encode()
		conn.Write(p)
	}

}

//discovery := packets.Discovery(uuid.New(), "simpsons.mp4")
//first, exists := n.Flooder.Flood(discovery)
//
//if exists {
//	utils.PrintStruct(first)
//} else {
//	log.Println("no response")
//}
