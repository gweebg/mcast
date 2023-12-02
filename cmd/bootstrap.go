package main

import (
	"flag"
	"github.com/gweebg/mcast/internal/bootstrap"
	"github.com/gweebg/mcast/internal/utils"
	"log"
)

func main() {

	address := flag.String("address", ":5001", "represents the listening address for the bootstrapper")
	config := flag.String("config", "config.json", "indicate relative path to configuration json file")
	flag.Parse()

	if *config == "config.json" {
		log.Printf("no configuration file specified, defaulting to '%v'\n", *config)
	}

	if *address == ":5001" {
		log.Printf("no address specified, defaulting to '%v'\n", *address)
	}

	bs := bootstrap.New(
		*config,
		*address,
	)

	utils.PrintStruct(bs.Config)
	bs.Listen()

}
