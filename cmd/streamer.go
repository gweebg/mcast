package main

import (
	"github.com/gweebg/mcast/internal/streamer"
	"time"
)

func main() {

	s := streamer.New(
		streamer.WithAddress("127.0.0.1:5000"),
		streamer.WithContentName("simpsons.mp4"),
	)

	go s.Stream()

	time.Sleep(30 * time.Second)

	s.Teardown()

}
