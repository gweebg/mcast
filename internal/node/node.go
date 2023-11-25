package node

import (
	"github.com/gweebg/mcast/internal/bootstrap"
	"github.com/gweebg/mcast/internal/handlers"
	"github.com/gweebg/mcast/internal/utils"
)

type Node struct {

	// information about what type of node I am, what are my neighbours
	Self bootstrap.Node

	// component responsible for flooding its neighbours with a packet
	Flooder Flooder

	// responsible for receiving tcp connections and handling them accordingly
	TCPHandler handlers.TCPConn
}

func New(bootstrapAddr string) *Node {

	self, err := setupSelf(bootstrapAddr)
	utils.Check(err)

	handler := handlers.NewTCP(
		handlers.WithListenTCP(),
		handlers.WithHandleTCP(),
	)

	return &Node{
		Self:       self,
		Flooder:    NewFlooder(self.Neighbours),
		TCPHandler: *handler,
	}
}
