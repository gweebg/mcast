package packets

import (
	"bytes"
	"encoding/gob"
	"github.com/google/uuid"
	"github.com/gweebg/mcast/internal/utils"
)

type Header struct {
	Flags     utils.FlagType
	RequestId uuid.UUID
	Source    string
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

	buf := bytes.NewBuffer(data)
	dec := gob.NewDecoder(buf)

	var p Packet

	if err := dec.Decode(&p); err != nil {
		return p, err
	}

	return p, nil
}

const (
	DISC  utils.FlagType = 0b1
	FOUND utils.FlagType = 0b10
	MISS  utils.FlagType = 0b100

	STREAM utils.FlagType = 0b1000
	PORT   utils.FlagType = 0b10000
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

func Found(requestId uuid.UUID, contentName string, source string) Packet {

	return Packet{
		Header: Header{
			Flags:     FOUND,
			RequestId: requestId,
			Hops:      0,
			Source:    source,
		},
		Payload: Payload{
			ContentName: contentName,
			Port:        "",
		},
	}

}

func Miss(requestId uuid.UUID, contentName string) Packet {

	return Packet{
		Header: Header{
			Flags:     MISS,
			RequestId: requestId,
			Hops:      0,
		},
		Payload: Payload{
			ContentName: contentName,
			Port:        "",
		},
	}

}

func Port(requestId uuid.UUID, contentName string, port string) Packet {

	return Packet{
		Header: Header{
			Flags:     PORT,
			RequestId: requestId,
			Hops:      0,
		},
		Payload: Payload{
			ContentName: contentName,
			Port:        port,
		},
	}

}

func Stream(requestId uuid.UUID, contentName string) Packet {

	return Packet{
		Header: Header{
			Flags:     STREAM,
			RequestId: requestId,
			Hops:      0,
		},
		Payload: Payload{
			ContentName: contentName,
			Port:        "",
		},
	}

}
