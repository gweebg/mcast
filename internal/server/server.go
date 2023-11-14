package server

import (
	"github.com/gweebg/mcast/internal/flags"
	"github.com/gweebg/mcast/internal/handlers"
	"github.com/gweebg/mcast/internal/packets"
	"github.com/gweebg/mcast/internal/utils"
)

type Packet = packets.BasePacket[string]

const (
	WAKE flags.FlagType = 0b1
	CONT flags.FlagType = 0b10
	CSND flags.FlagType = 0b100
	STOP flags.FlagType = 0b1000
)

type Server struct {
	Address string
	Config  Config

	TCPHandler handlers.TCPConn
}

func NewServer(addr, path string) *Server {

	tcp := handlers.NewTCP(
		handlers.WithListenTCP( /* can be the default */ ),
		handlers.WithHandleTCP( /* create handler */ ),
	) // tcp handler for this server

	return &Server{
		addr,
		utils.MustParseJson[Config](path, ValidateConfig),
		*tcp,
	}

}
