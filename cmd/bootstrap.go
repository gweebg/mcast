package main

import (
	"github.com/gweebg/mcast/internal/bootstrap"
	"github.com/gweebg/mcast/internal/utils"
)

func main() {

	bs := bootstrap.New(
		"./docs/example.json",
		":20001",
	)

	utils.PrintStruct(bs.Config)
	bs.Listen()

}
