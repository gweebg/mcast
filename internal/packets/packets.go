package packets

import (
	"bytes"
	"encoding/gob"

	"github.com/gweebg/mcast/internal/flags"
	"github.com/gweebg/mcast/internal/server"
)

type PacketHeader struct {
	Flag flags.FlagType
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

func Wake() BasePacket[string] {
	return BasePacket[string]{
		Header: PacketHeader{
			Flag: server.WAKE,
		},
	}
}

func Request(contentName string) BasePacket[string] {
	return BasePacket[string]{
		Header: PacketHeader{
			Flag: server.REQ,
		},
		Payload: contentName,
	}
}

func Ok() BasePacket[string] {
	return BasePacket[string]{
		Header: PacketHeader{
			Flag: server.OK,
		},
	}
}

func Stop(contentName string) BasePacket[string] {
	return BasePacket[string]{
		Header: PacketHeader{
			Flag: server.STOP,
		},
		Payload: contentName,
	}
}
