package main

import (
	svr "github.com/gweebg/mcast/internal/server"
	"log"
	"os"
	"os/signal"
)

func main() {

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)

	log.Println("Server is up and running!")

	var server *svr.Server = svr.NewServer("udp", ":1053")

	go func() {
		for range c {
			close(server.QuitCh)
			log.Fatal("Shutting down!")
		}
	}()

	server.Listen()

}
