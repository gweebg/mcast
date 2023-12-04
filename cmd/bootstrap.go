package main

import (
	"flag"
	"github.com/gweebg/mcast/internal/bootstrap"
	"github.com/gweebg/mcast/internal/utils"
	"log"
)

func main() {

	config := flag.String("config", "config.json", "indicate relative path to configuration json file")
	flag.Parse()

	if *config == "config.json" {
		log.Printf("no configuration file specified, defaulting to '%v'\n", *config)
	}

	bs := bootstrap.New(
		*config,
	)

	utils.PrintStruct(bs.Config)
	bs.Listen()

}
