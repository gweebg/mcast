package server

import (
	"errors"
	"github.com/gweebg/mcast/internal/utils"
	"log"
	"net"
)

type StreamingConn struct {
	Content     string
	Streamer    net.PacketConn // udp connection
	IsStreaming bool           // if the udp connection is active and streaming content
	Stopch      chan struct{}  // close channel for the udp streaming goroutine
}

func (s *StreamingConn) Start() {

	// open the file
	// start transmitting the data through the Streamer object
	// set IsStreaming to true

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

func (c *Connection) CreateStreamer(addr string, contentName string) (*StreamingConn, error) {

	// check if the content is already being streamed
	_, exists := c.StreamingPool[contentName]
	if exists {
		return nil, errors.New("content " + contentName + " is already being streamed")
	}

	// create the udp connection for streaming
	udpConn, err := net.ListenPacket("udp", addr)
	utils.Check(err)

	// create the streaming object
	cs := StreamingConn{
		Content:     contentName,
		Streamer:    udpConn,
		IsStreaming: false,
		Stopch:      make(chan struct{}),
	}
	c.StreamingPool[contentName] = cs

	return &cs, nil
}

func (c *Connection) StopStreaming(contentName string) error {

	conn, exists := c.StreamingPool[contentName]
	if !exists {
		return errors.New("content " + " is not being streamed")
	}

	conn.IsStreaming = false
	conn.Stopch <- struct{}{} // signal the streamer goroutine to stop streaming and close connection

	err := conn.Streamer.Close()
	utils.Check(err)

	return nil
}

func (c *Connection) Shutdown() {

	// closing the tcp connection
	err := c.Conn.Close()
	if err != nil {
		log.Printf("(%v) connection is already closed, skipping", c.Conn.RemoteAddr().String())
	}

	// closing every udp connection
	for key, _ := range c.StreamingPool {
		err = c.StopStreaming(key)
		if err != nil {
			log.Printf("(%v) content is not streaming, skippin", c.Conn.RemoteAddr().String())
		}
	}
}
