package databases

import (
	"errors"
	"github.com/gweebg/mcast/internal/streamer"
	"sync"
)

type RelayDb struct {
	Pool map[string]*streamer.Relay

	poolLock sync.RWMutex

	currentPort uint64
}

func NewRelayDb(initialPort int) *RelayDb {

	return &RelayDb{
		Pool:        make(map[string]*streamer.Relay),
		currentPort: uint64(initialPort),
	}
}

func (r *RelayDb) IsStreaming(content string) bool {

	r.poolLock.RLock()
	defer r.poolLock.RUnlock()

	_, exists := r.Pool[content]
	return exists

}

func (r *RelayDb) GetRelay(content string) (*streamer.Relay, bool) {

	r.poolLock.RLock()
	defer r.poolLock.RUnlock()

	relay, exists := r.Pool[content]
	return relay, exists

}

func (r *RelayDb) SetRelay(content string, relay *streamer.Relay) error {

	r.poolLock.Lock()
	defer r.poolLock.Unlock()

	_, exists := r.Pool[content]
	if exists {
		return errors.New("relay already exists for content " + content)
	}

	r.Pool[content] = relay
	return nil
}

func (r *RelayDb) NextPort() uint64 {
	port := r.currentPort
	r.currentPort++
	return port
}
