package packets

import (
	"bytes"
	"encoding/gob"
	"github.com/gweebg/mcast/internal/flags"

	"github.com/google/uuid"
)

type Header struct {
	Flags     flags.FlagType
	RequestId uuid.UUID
	Hops      uint64
}

type Payload struct {
	ContentName string
	Port        string
}

type Packet struct {
	Header  Header
	Payload Payload
}

func (p Packet) Encode() ([]byte, error) {

	buf := new(bytes.Buffer) // using bytes.Buffer because implements io.Writer/Reader
	enc := gob.NewEncoder(buf)

	if err := enc.Encode(p); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil // returning the bytes stored in the buffer and no error
}

func DecodePacket(data []byte) (Packet, error) {

	// todo: bloated, make this as much generic as possible

	buf := bytes.NewBuffer(data)
	dec := gob.NewDecoder(buf)

	var p Packet

	if err := dec.Decode(&p); err != nil {
		return p, err
	}

	return p, nil
}

const (
	DISC flags.FlagType = 0b1
)

func Discovery(requestId uuid.UUID, contentName string) Packet {

	return Packet{
		Header: Header{
			Flags:     DISC,
			RequestId: requestId,
			Hops:      0,
		},
		Payload: Payload{
			ContentName: contentName,
			Port:        "",
		},
	}
}