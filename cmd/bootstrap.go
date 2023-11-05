package main

import (
	"github.com/gweebg/mcast/internal/bootstrap"
)

func main() {

	bs := bootstrap.New(
		"./docs/example.json",
		":20001",
	)

	bs.Listen()

}
