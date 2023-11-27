package node

import (
	"errors"
	"github.com/google/uuid"
	"github.com/gweebg/mcast/internal/bootstrap"
	"github.com/gweebg/mcast/internal/packets"
	"github.com/gweebg/mcast/internal/utils"
	"net"
	"sync"
)

const ReadSize = 1024

var (
	Request = packets.BasePacket[string]{
		Header: packets.PacketHeader{
			Flag: bootstrap.GET,
		},
		Payload: "",
	}
)

func setupSelf(bootstrapAddr string) (bootstrap.Node, error) {

	tcpAddr, err := net.ResolveTCPAddr("tcp", bootstrapAddr)
	utils.Check(err)

	conn, err := net.DialTCP("tcp", nil, tcpAddr)
	utils.Check(err)

	p, err := packets.Encode[string](Request)
	utils.Check(err)

	_, err = conn.Write(p)
	utils.Check(err)

	buffer := make([]byte, ReadSize)
	size, err := conn.Read(buffer)
	utils.Check(err)

	response, err := packets.Decode[bootstrap.Node](buffer[:size])
	utils.Check(err)

	if response.Header.Flag.OnlyHasFlag(bootstrap.SEND) {
		return response.Payload, nil
	}

	return bootstrap.Node{}, errors.New("expected flag SEND from bootstrapper, but received another")
}

/* ----------------------------------------------------------------------------- */

type RequestDb struct {
	data map[uuid.UUID]bool
	mu   sync.RWMutex
}

func NewRequestDb() *RequestDb {
	return &RequestDb{
		data: make(map[uuid.UUID]bool),
	}
}

func (r *RequestDb) IsHandled(id uuid.UUID) bool {

	r.mu.RLock()
	defer r.mu.RUnlock()

	_, exists := r.data[id]
	return exists
}

func (r *RequestDb) Set(id uuid.UUID, state bool) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.data[id] = state
}
