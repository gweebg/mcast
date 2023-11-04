package main

import (
	"bufio"
	"fmt"
	"log"
	"net"
)

func main() {

	addr, err := net.ResolveUDPAddr("udp", "127.0.0.1:1053")
	if err != nil {
		log.Fatal("Could not parse UDP address.")
	}

	conn, err := net.DialUDP("udp", nil, addr)
	if err != nil {
		log.Fatalf("Could not connect to %s\n", addr.String())
	}

	_, err = conn.Write([]byte("Ping"))
	if err != nil {
		log.Fatal("Could not write to server.")
	}

	data, err := bufio.NewReader(conn).ReadString('\n')
	if err != nil {
		fmt.Println(err)
		return
	}

	fmt.Print("> ", string(data))

}
