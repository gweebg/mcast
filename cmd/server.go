package main

import (
	"flag"
	"github.com/gweebg/mcast/internal/server"
	"log"
)

func main() {

	address := flag.String("address", "", "address:port string to listen for tcp connections")
	config := flag.String("config", "server_config.json", "configuration file for the server")

	flag.Parse()

	if *address == "" {
		log.Fatalf("server address:port is mandatory\n")
	}

	if *config == "server_config.json" {
		log.Printf("config file not specified, defaulting to '%v'\n", *config)
	}

	srv := server.New(*address, *config)
	srv.Run()

}
