package server

import (
	"github.com/gweebg/mcast/internal/flags"
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
}

func NewServer(addr, path string) *Server {

	return &Server{
		addr,
		utils.MustParseJson[Config](path, ValidateConfig),
	}

}

//
//func (s *Server) serve(pc net.PacketConn, addr net.Addr, buf []byte) {
//
//	incoming := string(buf[:])
//	log.Printf("%s> %s\n", addr.String(), strings.Replace(incoming, "\n", "", -1))
//
//	if strings.TrimRight(incoming, "\n") == "Ping" {
//
//		to, err := pc.WriteTo([]byte(s.DefaultMessage), addr)
//		if err != nil {
//			log.Printf("Failed to write %d bytes to %s\n", to, addr.String())
//			return
//		}
//
//		log.Printf("server> Sent 'pong' to %s\n", addr.String())
//
//	} else if strings.TrimRight(incoming, "\n") == "Quit" {
//
//		log.Printf("server> Shutting down server!")
//		close(s.QuitCh)
//
//	}
//
//}
//
//func (s *Server) Listen() {
//
//	pc, err := net.ListenPacket(s.Protocol, s.Address)
//	if err != nil {
//		log.Fatal(err)
//	}
//
//	defer func(pc net.PacketConn) {
//		err := pc.Close()
//		if err != nil {
//			log.Fatal("Could not close close connection.")
//		}
//	}(pc)
//
//	for {
//
//		select {
//		case <-s.QuitCh:
//			return
//		default:
//		}
//
//		buf := make([]byte, 1024)
//		n, addr, err := pc.ReadFrom(buf)
//
//		if err != nil {
//			log.Printf("Failed to read %d bytes from '%s'\n", n, addr.String())
//			continue
//		}
//
//		go s.serve(pc, addr, buf[:n])
//
//	}
//
//}
