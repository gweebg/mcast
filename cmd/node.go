package main

import (
	"github.com/google/uuid"
	"github.com/gweebg/mcast/internal/node"
	"github.com/gweebg/mcast/internal/packets"
	"github.com/gweebg/mcast/internal/utils"
	"log"
)

func main() {

	bootstrapAddr := "127.0.0.1:20001"
	n := node.New(bootstrapAddr)

	utils.PrintStruct(n.Self)

	discovery := packets.Discovery(uuid.New(), "simpsons.mp4")
	first, exists := n.Flooder.Flood(discovery)

	if exists {
		utils.PrintStruct(first)
	} else {
		log.Println("no response")
	}
}
