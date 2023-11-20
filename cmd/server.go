package main

import (
	"github.com/gweebg/mcast/internal/server"
)

func main() {

	srv := server.New("127.0.0.1:20010", "./docs/server_config.json")
	srv.Run()

}
