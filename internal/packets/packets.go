package packets

import (
	"bytes"
	"encoding/gob"

	"github.com/gweebg/mcast/internal/flags"
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

const (
	WAKE flags.FlagType = 0b1
	STOP flags.FlagType = 0b1000
	OK   flags.FlagType = 0b10000
	REQ  flags.FlagType = 0b100000

	PING flags.FlagType = 0b1000000

	CONT flags.FlagType = 0b10
	CSND flags.FlagType = 0b100

	SDP flags.FlagType = 0b100000
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

func SendSdp(sdp []byte) BasePacket[[]byte] {
	return BasePacket[[]byte]{
		Header: PacketHeader{
			Flag: SDP,
		},
		Payload: sdp,
	}
}
