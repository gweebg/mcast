package databases

import (
	"github.com/google/uuid"
	"sync"
)

type RequestDb struct {
	Requests    map[uuid.UUID]bool
	requestLock sync.RWMutex
}

func NewRequestDb() *RequestDb {
	return &RequestDb{
		Requests: make(map[uuid.UUID]bool),
	}
}

func (r *RequestDb) IsHandled(id uuid.UUID) bool {

	r.requestLock.RLock()
	defer r.requestLock.RUnlock()

	_, exists := r.Requests[id]
	return exists
}

func (r *RequestDb) Set(id uuid.UUID, state bool) {
	r.requestLock.Lock()
	defer r.requestLock.Unlock()

	r.Requests[id] = state
}
