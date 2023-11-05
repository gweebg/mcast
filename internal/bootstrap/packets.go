package bootstrap

import (
	"bytes"
	"encoding/gob"
	"net"
)

type FlagType uint8

const (
	GET FlagType = iota
	SND
	ERR
)

type PacketHeader struct {
	Flag  FlagType
	Count uint
}

type Packet struct {
	Header  PacketHeader
	Payload []net.IP
}

func (p Packet) Encode() ([]byte, error) {

	buf := new(bytes.Buffer) // using bytes.Buffer because implements io.Writer/Reader
	enc := gob.NewEncoder(buf)

	if err := enc.Encode(p); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil // returning the bytes stored in the buffer and no error

}

func Decode(data []byte) (Packet, error) {

	buf := bytes.NewBuffer(data)
	dec := gob.NewDecoder(buf)

	var p Packet

	if err := dec.Decode(&p); err != nil {
		return p, err
	}

	return p, nil

}
