package bootstrap

import (
	"net/netip"
)

type NodeType string

const (
	Client          NodeType = "client"
	Server          NodeType = "server"
	RendezvousPoint NodeType = "rendezvous"
	ONode           NodeType = "node"
)

type Node struct {
	Type       NodeType         `json:"type"`
	SelfPort   uint16           `json:"selfPort"`
	Neighbours []netip.AddrPort `json:"neighbours"`
}

type Nodes map[string]Node

type Config struct {
	NodeGroup Nodes `json:"nodes"`
}

func ValidateConfig(c Config) bool {

	for _, v := range c.NodeGroup {
		if !(v.Type == Client || v.Type == Server || v.Type == RendezvousPoint || v.Type == ONode) {
			return false
		}
	}
	return true
}
