package server

import "github.com/gweebg/mcast/internal/packets"

func ContentInfoPacket(c []ConfigItem) packets.BasePacket[[]ConfigItem] {

	return packets.BasePacket[[]ConfigItem]{
		Header:  packets.PacketHeader{Flag: packets.CONT},
		Payload: c,
	}
}

func ContentPortPacket(port string) packets.BasePacket[string] {

	return packets.BasePacket[string]{
		Header:  packets.PacketHeader{Flag: packets.CSND},
		Payload: port,
	}
}

func Pong() packets.BasePacket[string] {

	return packets.BasePacket[string]{
		Header:  packets.PacketHeader{Flag: packets.PING},
		Payload: "pong",
	}
}
