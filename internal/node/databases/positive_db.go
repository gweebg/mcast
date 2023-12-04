package databases

import (
	"errors"
	"github.com/google/uuid"
	"sync"
)

type PositiveDb struct {
	Positives     map[uuid.UUID]string
	positivesLock sync.RWMutex
}

func NewPositiveDb() *PositiveDb {

	return &PositiveDb{
		Positives: make(map[uuid.UUID]string),
	}
}

func (p *PositiveDb) CheckAndGet(id uuid.UUID) (string, bool) {

	p.positivesLock.RLock()
	defer p.positivesLock.RUnlock()

	relay, exists := p.Positives[id]
	return relay, exists

}

func (p *PositiveDb) Set(id uuid.UUID, address string) error {

	p.positivesLock.Lock()
	defer p.positivesLock.Unlock()

	_, exists := p.Positives[id]
	if exists {
		return errors.New("address already exists for id " + id.String())
	}

	p.Positives[id] = address
	return nil
}
