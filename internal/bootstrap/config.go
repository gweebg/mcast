package bootstrap

import (
	"github.com/gweebg/mcast/internal/utils"
	"net/netip"
)

type NodeType string

const (
	RendezvousPoint NodeType = "rendezvous"
	ONode           NodeType = "node"
	Bootstrapper    NodeType = "bootstrap"
)

const (
	SEND utils.FlagType = 0b1
	ERR  utils.FlagType = 0b10
	GET  utils.FlagType = 0b100
)

type Node struct {

	// Type - type of the node, specified by 'type' in json.
	Type NodeType `json:"type"`

	// SelfIp - ip:port for the current node, where to listen.
	SelfIp string `json:"self"`

	// Neighbours - set of neighbour nodes if node is 'node', or set of available servers if node is 'rendezvous'.
	Neighbours []netip.AddrPort `json:"neighbours"`
}

type Nodes map[string]Node

type Config struct {
	// NodeGroup - main json group that contains every node on the network, with every interface specified.
	NodeGroup Nodes `json:"nodes"`

	// Bootstrapper - representation of this Bootstrap node.
	Bootstrapper Node `json:"bootstrapper"`
}

func ValidateConfig(c Config) bool {

	for _, v := range c.NodeGroup {
		if !(v.Type == Bootstrapper || v.Type == RendezvousPoint || v.Type == ONode) {
			return false
		}
	}
	return true
}
