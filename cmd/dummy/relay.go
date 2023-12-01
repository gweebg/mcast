package main

import (
	"github.com/gweebg/mcast/internal/node"
	"github.com/gweebg/mcast/internal/utils"
)

func main() {

	r := node.NewRelay("video.mp4", "127.0.0.1:5000", "5001")

	err := r.Add("127.0.0.1:5001")
	utils.Check(err)

	err = r.Add("127.0.0.1:5002")
	utils.Check(err)

	go r.Loop()

	select {}
	//time.Sleep(5 * time.Second)
	//
	//err = r.Add("127.0.0.1:5003")
	//utils.Check(err)
	//
	//err = r.Add("127.0.0.1:5004")
	//utils.Check(err)

}
