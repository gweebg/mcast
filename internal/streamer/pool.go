package streamer

import (
	"errors"
	"sync"
)

type ContentMap map[string]*Streamer

func (s ContentMap) AddEntry(stmr *Streamer) error {

	_, exists := s[stmr.ContentName]
	if !exists {
		s[stmr.ContentName] = stmr
		return nil
	}

	return errors.New("streamer already exists for the video " + stmr.ContentName + " on the address " + stmr.Address)

}

type StreamingPool struct {
	Pool map[string]ContentMap // { address : { videoName: Streamer, ... }, ... }
	mu   sync.RWMutex
}

func NewStreamingPool() StreamingPool {
	return StreamingPool{
		Pool: make(map[string]ContentMap),
	}
}

func (p *StreamingPool) Add(addr string, s *Streamer) error {

	p.mu.Lock()
	defer p.mu.Unlock()

	streamers, exists := p.Pool[addr]
	if exists {
		err := streamers.AddEntry(s)
		return err
	}

	err := p.Pool[addr].AddEntry(s)
	return err

}

func (p *StreamingPool) GetPool(addr string) (ContentMap, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	streamers, exists := p.Pool[addr]
	if !exists {
		return nil, errors.New("address " + addr + " not found")
	}

	return streamers, nil
}

func (p *StreamingPool) GetContent(addr string, content string) (*Streamer, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	streamers, exists := p.Pool[addr]
	if !exists {
		return nil, errors.New("address " + addr + " not found")
	}

	stmr, exists := streamers[content]
	if !exists {
		return nil, errors.New("content " + content + " not found at the address " + addr)
	}

	return stmr, nil
}

func (p *StreamingPool) Delete(addr string, content string) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	streamers, exists := p.Pool[addr]
	if !exists {
		return errors.New("address " + addr + " not found")
	}

	stmr, exists := streamers[content]
	if !exists {
		return errors.New("content " + content + " not found at the address " + addr)
	}

	stmr.Teardown()

	delete(streamers, content)
	return nil
}
