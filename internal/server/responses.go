package server

import "github.com/gweebg/mcast/internal/packets"

func ContentInfoPacket(c []ConfigItem) packets.BasePacket[[]ConfigItem] {

	return packets.BasePacket[[]ConfigItem]{
		Header:  packets.PacketHeader{Flag: CONT},
		Payload: c,
	}
}

func ContentPortPacket(p int) packets.BasePacket[int] {

	return packets.BasePacket[int]{
		Header:  packets.PacketHeader{Flag: CSND},
		Payload: p,
	}
}
