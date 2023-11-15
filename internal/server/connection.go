package server

import (
	"errors"
	"net"
)

type StreamingConn struct {
	Streamer    net.PacketConn // udp connection
	IsStreaming bool           // if the udp connection is active and streaming content
	Stopch      chan struct{}  // close channel for the udp streaming goroutine
}

type Connection struct {
	Conn          net.Conn                 // tcp connection
	StreamingPool map[string]StreamingConn // streaming active connections
}

func NewConnection(conn net.Conn) *Connection {
	return &Connection{
		Conn:          conn,
		StreamingPool: make(map[string]StreamingConn),
	}
}

func (c *Connection) CreateStreamer(conn net.PacketConn, contentName string) error {
	// todo: passing already the conn ? should be the addr:port ?

	_, exists := c.StreamingPool[contentName]
	if exists {
		return errors.New("content " + contentName + " is already being streamed")
	}

	c.StreamingPool[contentName] = StreamingConn{
		Streamer:    conn,
		IsStreaming: false,
		Stopch:      make(chan struct{}),
	}

	return nil
}

func (c *Connection) StopStreaming(contentName string) error {

	conn, exists := c.StreamingPool[contentName]
	if !exists {
		return errors.New("content " + " is not being streamed")
	}

	conn.IsStreaming = false
	conn.Stopch <- struct{}{} // signal the streamer goroutine to stop streaming and close connection

	return nil
}
