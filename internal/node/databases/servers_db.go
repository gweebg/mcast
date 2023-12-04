package databases

import (
	"github.com/gweebg/mcast/internal/packets"
	"github.com/gweebg/mcast/internal/server"
	"github.com/gweebg/mcast/internal/utils"
	"log"
	"net"
	"sync"
	"time"
)

type ServerInfo struct {
	// Address - the address where the node will talk to the server.
	Address string

	// Latency - current latency of the server.
	Latency        int64
	latencyHistory []int64

	// Jitter - current jitter of the server.
	Jitter      float64
	prevLatency float64

	// metricLock - lock to prevent race conditions.
	metricLock sync.Mutex

	// Content - slice containing which content the server has available to stream.
	Content []server.ConfigItem

	// Streaming - slice containing the content that is being streamed at the moment.
	Streaming []string

	// Conn - tcp connection to the server at Address.
	Conn *net.TCPConn

	// Ticker - used to send metric packets once every 5 seconds.
	Ticker *time.Ticker
	// TickerChan - needed to stop the ticker on demand.
	TickerChan chan bool
}

func (s *ServerInfo) SetStreaming(contentName string) {

	s.metricLock.Lock()
	defer s.metricLock.Unlock()

	s.Streaming = append(s.Streaming, contentName)
}

func (s *ServerInfo) UnsetStreaming(contentName string) {

	s.metricLock.Lock()
	defer s.metricLock.Unlock()

	filtered := make([]string, 0)
	for _, content := range s.Streaming {
		if content != contentName {
			filtered = append(filtered, content)
		}
	}

	s.Streaming = filtered
}

func (s *ServerInfo) Measure() {

	remote := s.Conn.RemoteAddr().String()

	s.Ticker = time.NewTicker(5 * time.Second)

	go func() {
		log.Printf("(metrics %v) started metrics loop\n", remote)
		for {
			select {

			case <-s.TickerChan:
				return

			case <-s.Ticker.C:

				ping := packets.Ping()

				packet, err := packets.Encode[string](ping)
				utils.Check(err)

				startTime := time.Now()

				_, err = s.Conn.Write(packet)
				utils.Check(err)

				_, err = s.Conn.Read(nil)
				utils.Check(err)

				stopTime := time.Now()

				elapsedTime := stopTime.Sub(startTime).Milliseconds()

				s.metricLock.Lock()

				s.latencyHistory = append(s.latencyHistory, elapsedTime)
				s.Latency = utils.SumSliceInt64(s.latencyHistory) / int64(len(s.latencyHistory))

				if s.prevLatency == -1 {
					s.Jitter = 1.0
				}

				s.Jitter = s.prevLatency / float64(s.Latency)

				s.metricLock.Unlock()
			}
		}
	}()

}

// CalculateMetrics calculates the general metrics of a server, the Latency is given
// a weight of 60% while the Jitter is given a weight of 40%. Packet loss is not accounted
// for the metrics calculation, yet.
func (s *ServerInfo) CalculateMetrics() float64 {
	return (float64(s.Latency)*0.6 + s.Jitter*0.4) / 100
}

type Servers struct {
	Data     map[string]*ServerInfo
	dataLock sync.RWMutex
}

// NewServers populate the Servers struct with the ServerInfo objects by passing
// their addresses.
func NewServers(addrs []string) *Servers {

	s := Servers{
		Data: make(map[string]*ServerInfo),
	}

	for _, addr := range addrs {
		s.Data[addr] = &ServerInfo{
			latencyHistory: make([]int64, 0),
			prevLatency:    -1,

			Address:    addr,
			Content:    make([]server.ConfigItem, 0),
			TickerChan: make(chan bool),
		}
	}

	return &s
}

func (s *Servers) GetServer(address string) (*ServerInfo, bool) {

	s.dataLock.RLock()
	defer s.dataLock.RUnlock()

	info, exists := s.Data[address]
	return info, exists

}

// ContentExists checks if a provided content is available in any of the active
// servers.
func (s *Servers) ContentExists(contentName string) bool {

	for _, srv := range s.Data {
		for _, content := range srv.Content {
			if content.Name == contentName {
				return true
			}
		}
	}

	return false
}

func contains(s []server.ConfigItem, str string) bool {
	for _, v := range s {
		if v.Name == str {
			return true
		}
	}
	return false
}

// GetBestServer returns the connection to the best server (better metric)
// with the content (contentName) available.
func (s *Servers) GetBestServer(contentName string) *ServerInfo {

	s.dataLock.RLock()
	defer s.dataLock.RUnlock()

	var best *ServerInfo

	for _, srv := range s.Data {

		if !contains(srv.Content, contentName) { // if server does not contentName, skip iteration
			continue
		}

		if best == nil || best.CalculateMetrics() < srv.CalculateMetrics() {
			best = srv
		}
	}
	return best
}

func (s *Servers) WhoIsStreaming(contentName string) (*ServerInfo, bool) {

	s.dataLock.RLock()
	defer s.dataLock.RUnlock()

	for _, serv := range s.Data {
		for _, content := range serv.Streaming {
			if content == contentName {
				return serv, true
			}
		}
	}

	return nil, false
}
