package server

import "github.com/gweebg/mcast/internal/packets"

func ContentInfoPacket(c []ConfigItem) packets.BasePacket[[]ConfigItem] {

	return packets.BasePacket[[]ConfigItem]{
		Header:  packets.PacketHeader{Flag: CONT},
		Payload: c,
	}
}

func ContentAccessPacket(p uint64) packets.BasePacket[uint64] {

	return packets.BasePacket[uint64]{
		Header:  packets.PacketHeader{Flag: CSND},
		Payload: p,
	}
}
