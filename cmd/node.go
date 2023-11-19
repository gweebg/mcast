package main

import "github.com/gweebg/mcast/internal/node"


func main() {
    // todo: change bootstrapAddr to arg
    bootstrapAddr := "127.0.0.1:20001"
    node := node.New(bootstrapAddr)
    node.Run()
}
