package main

import (
	"flag"
	"github.com/gweebg/mcast/internal/node"
	"github.com/gweebg/mcast/internal/utils"
	"log"
	"net/netip"
)

func main() {

	bootstrapper := flag.String("bootstrap", "", "address of the bootstrapper node")

	flag.Parse()

	if *bootstrapper == "" {
		log.Fatalf("bootstrapper address is mandatory\n")
	}

	_, err := netip.ParseAddrPort(*bootstrapper)
	utils.Check(err)

	onode := node.New(*bootstrapper)
	onode.Run()
}
