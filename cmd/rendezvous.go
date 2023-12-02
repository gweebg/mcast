package main

import (
	"flag"
	"log"
	"net/netip"

	"github.com/gweebg/mcast/internal/rendezvous"
	"github.com/gweebg/mcast/internal/utils"
)

type arrayFlags []string

func (i *arrayFlags) String() string {
	return "my string representation"
}

func (i *arrayFlags) Set(value string) error {
	*i = append(*i, value)
	return nil
}

func main() {

	var servers arrayFlags

	address := flag.String("address", "", "address of the rendezvous node")
	flag.Var(&servers, "server", "list of server address:port for the rendezvous node")

	flag.Parse()

	if *address == "" {
		log.Fatalf("rendezvous address is mandatory\n")
	}

	addr, err := netip.ParseAddrPort(*address)
	utils.Check(err)

	addrString := addr.Addr().String()
	if addrString == "127.0.0.1" || addrString == "localhost" {
		log.Fatalf("rendezvous address cannot be localhost|127.0.0.1\n")
	}

	rend := rendezvous.New(*address, servers...)
	rend.Run()

}
