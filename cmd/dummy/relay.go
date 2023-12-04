package main

import (
	utils2 "github.com/gweebg/mcast/internal/node/utils"
	"github.com/gweebg/mcast/internal/utils"
	"log"
)

func main() {

	list := make([]*utils2.Relay, 0)
	r := utils2.NewRelay("video.mp4", "127.0.0.1:5000", "5001")

	err := r.Add("127.0.0.1:5001")
	utils.Check(err)

	err = r.Add("127.0.0.1:5002")
	utils.Check(err)

	list = append(list, r)

	log.Println(list)

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
