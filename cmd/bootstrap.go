package main

import (
	"flag"
	"github.com/gweebg/mcast/internal/bootstrap"
	"github.com/gweebg/mcast/internal/utils"
	"log"
)

func main() {

	configFlag := flag.String("config", "config.json", "indicate relative path to configuration json file")
	flag.Parse()

	if *configFlag == "config.json" {
		log.Print("no configuration file specified, defaulting to 'config.json'")
	}

	bs := bootstrap.New(
		*configFlag,
		":5001",
	)

	utils.PrintStruct(bs.Config)
	bs.Listen()

}
