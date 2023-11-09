package bootstrap

import (
	"encoding/json"
	"errors"
	"log"
	"net/netip"
	"os"
)

type NodeType string

const (
	Client          NodeType = "client"
	Server          NodeType = "server"
	RendezvousPoint NodeType = "rendezvous"
)

type Node struct {
	Type       NodeType         `json:"type"`
	SelfPort   uint16           `json:"selfPort"`
	Neighbours []netip.AddrPort `json:"neighbours"`
}

type Nodes map[netip.Addr]Node

type Config struct {
	NodeGroup Nodes `json:"nodes"`
}

func MustParse(filename string) Config {

	data, err := os.ReadFile(filename)
	if err != nil {
		log.Fatalf("Could not read file '%s'\n", filename)
	}

	var cfg Config
	err = json.Unmarshal(data, &cfg)
	if err != nil {
		log.Fatal(err)
	}

	err = checkConfig(cfg)
	if err != nil {
		log.Fatal(err)
	}

	return cfg
}

func checkConfig(c Config) error {

	for _, v := range c.NodeGroup {
		if !(v.Type == Client || v.Type == Server || v.Type == RendezvousPoint) {
			return errors.New("invalid 'type' value, type must be client | rendezvous | server")
		}
	}
	return nil
}
