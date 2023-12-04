package packets

import (
	"bytes"
	"encoding/gob"
	"github.com/gweebg/mcast/internal/utils"
)

type PacketHeader struct {
	Flag utils.FlagType
}

type BasePacket[T any] struct {
	Header  PacketHeader
	Payload T
}

func Encode[T any](p BasePacket[T]) ([]byte, error) {

	buf := new(bytes.Buffer) // using bytes.Buffer because implements io.Writer/Reader
	enc := gob.NewEncoder(buf)

	if err := enc.Encode(p); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil // returning the bytes stored in the buffer and no error

}

func Decode[T any](data []byte) (BasePacket[T], error) {

	buf := bytes.NewBuffer(data)
	dec := gob.NewDecoder(buf)

	var p BasePacket[T]

	if err := dec.Decode(&p); err != nil {
		return p, err
	}

	return p, nil

}

const (
	WAKE utils.FlagType = 0b1
	CONT utils.FlagType = 0b10
	CSND utils.FlagType = 0b100
	STOP utils.FlagType = 0b1000
	OK   utils.FlagType = 0b10000
	REQ  utils.FlagType = 0b100000
	PING utils.FlagType = 0b1000000
)

func Wake() BasePacket[string] {
	return BasePacket[string]{
		Header: PacketHeader{
			Flag: WAKE,
		},
	}
}

func Request(contentName string) BasePacket[string] {
	return BasePacket[string]{
		Header: PacketHeader{
			Flag: REQ,
		},
		Payload: contentName,
	}
}

func Ok() BasePacket[string] {
	return BasePacket[string]{
		Header: PacketHeader{
			Flag: OK,
		},
	}
}

func Stop(contentName string) BasePacket[string] {
	return BasePacket[string]{
		Header: PacketHeader{
			Flag: STOP,
		},
		Payload: contentName,
	}
}

func Ping() BasePacket[string] {
	return BasePacket[string]{
		Header: PacketHeader{
			Flag: PING,
		},
		Payload: "hello!",
	}
}
