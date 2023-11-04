package main

import (
	"github.com/gweebg/mcast/internal/bootstrap"
	"log"
)

func main() {
	bs := bootstrap.New("./docs/example.json")
	log.Printf("%v\n", bs.Neighbours)
}
