package server

import (
	"net"
)

type StreamingConn struct {
	Streamer    net.PacketConn // udp connection
	IsStreaming bool           // if the udp connection is active and streaming content
	Stopch      chan struct{}  // close channel for the udp streaming goroutine
}

type Connection struct {
	Conn     net.Conn // tcp connection
	Streamer StreamingConn
}

func NewConnection(conn net.Conn) *Connection {
	return &Connection{
		Conn: conn,
		Streamer: StreamingConn{
			Streamer:    nil,
			IsStreaming: false,
			Stopch:      make(chan struct{}),
		},
	}
}
