package main

import (
	"github.com/gweebg/mcast/internal/packets"
	"github.com/gweebg/mcast/internal/server"
	"github.com/gweebg/mcast/internal/utils"
	"io"
	"net"
	"os"
	"os/exec"
	"strconv"
	"time"
)

var (
	WakePacket packets.BasePacket[string] = packets.BasePacket[string]{
		Header: packets.PacketHeader{
			Flag: server.WAKE,
		},
	}

	ReqPacket packets.BasePacket[string] = packets.BasePacket[string]{
		Header: packets.PacketHeader{
			Flag: server.REQ,
		},
		Payload: "simpsons.mp4",
	}

	OkPacket packets.BasePacket[string] = packets.BasePacket[string]{
		Header: packets.PacketHeader{
			Flag: server.OK,
		},
	}

	StopPacket packets.BasePacket[string] = packets.BasePacket[string]{
		Header: packets.PacketHeader{
			Flag: server.STOP,
		},
		Payload: "simpsons.mp4",
	}
)

func main() {
	servAddr := "127.0.0.1:7000"

	tcpAddr, err := net.ResolveTCPAddr("tcp", servAddr)
	utils.Check(err)

	conn, err := net.DialTCP("tcp", nil, tcpAddr)
	utils.Check(err)

	/* Wake Packet */

	p, err := packets.Encode[string](WakePacket)
	utils.Check(err)

	_, err = conn.Write(p)
	utils.Check(err)

	buffer := make([]byte, 1024)
	n, err := conn.Read(buffer)
	utils.Check(err)

	recv, err := packets.Decode[[]server.ConfigItem](buffer[:n])
	utils.Check(err)

	utils.PrintStruct(recv)

	/* Req Packet */

	p, err = packets.Encode[string](ReqPacket)
	utils.Check(err)

	_, err = conn.Write(p)
	utils.Check(err)

	n, err = conn.Read(buffer)
	utils.Check(err)

	resp, err := packets.Decode[int](buffer[:n])
	utils.Check(err)

	utils.PrintStruct(resp)

	/* Ok Packet */

	streamingPort := strconv.FormatInt(int64(resp.Payload), 10)
	streamingAddr := "127.0.0.1" + ":" + streamingPort

	addr, err := net.ResolveUDPAddr("udp", streamingAddr)
	utils.Check(err)

	udpConn, err := net.ListenUDP("udp", addr)
	utils.Check(err)

	defer conn.Close()

	// start ffplay command
	ffplayCmd := exec.Command("ffplay", "-f", "mpegts", "-")
	ffplayStdin, err := ffplayCmd.StdinPipe()
	utils.Check(err)

	ffplayCmd.Stdout = os.Stdout
	ffplayCmd.Stderr = os.Stderr

	// Start ffplay process
	err = ffplayCmd.Start()
	utils.Check(err)

	// Goroutine for continuously reading and writing MPEG TS packets to ffplay
	go func() {
		buffer := make([]byte, 188*10) // 188 bytes per MPEG TS packet
		for {
			n, _, err := udpConn.ReadFromUDP(buffer)
			if err != nil {
				if err != io.EOF {
					panic(err)
				}
				break
			}

			_, err = ffplayStdin.Write(buffer[:n])
			if err != nil {
				panic(err)
			}
		}
	}()

	time.Sleep(100 * time.Millisecond)

	p, err = packets.Encode[string](OkPacket)
	utils.Check(err)

	_, err = conn.Write(p)
	utils.Check(err)

	err = ffplayCmd.Wait()

}
