package rendezvous

import (
	"github.com/gweebg/mcast/internal/server"
	"net"
	"sync"
	"time"
)

type Servers map[string]*ServerInfo

// NewServers populate the Servers struct with the ServerInfo objects by passing
// their addresses.
func NewServers(addrs []string) Servers {
	srvs := make(Servers)

	for _, addr := range addrs {
		srvs[addr] = &ServerInfo{
			Address:    addr,
			Content:    make([]server.ConfigItem, 0),
			tickerChan: make(chan bool),
		}
	}

	return srvs
}

type ServerInfo struct {

	// the address where the node will talk to the server.
	Address string

	// packet loss of the server.
	PacketLoss uint64
	// Jitter of the server.
	Jitter float32
	// Latency of the server.
	Latency float32
	// Metric lock to prevent race conditions.
	mMu sync.Mutex

	// which content the server has available.
	Content []server.ConfigItem
	// tcp connection to the server at Address.
	Conn *net.TCPConn

	// used to send metric packets once every 5 seconds
	Ticker *time.Ticker
	// needed to stop the ticker on teardown
	tickerChan chan bool
}

// CalculateMetrics calculates the general metrics of a server, the Latency is given
// a weight of 60% while the Jitter is given a weight of 40%. Packet loss is not accounted
// for the metrics calculation, yet.
func (s *ServerInfo) CalculateMetrics() float32 {
	return ((s.Latency * 0.6) + (s.Jitter * 0.4)) / 100
}
