package main

import (
	"flag"
	"log"
	"net/netip"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/gweebg/mcast/internal/packets"
	"github.com/gweebg/mcast/internal/utils"
)

func SetSdpPort(originalSDP, port string) string {
	updatedSDP := strings.ReplaceAll(originalSDP, "????", port)
	return updatedSDP
}

func Listen(sdp []byte, content string) {

	sdpFilename := "sdp_" + content + ".sdp" + time.Now().String()

	tempFile, err := os.CreateTemp("", sdpFilename)
	utils.Check(err)

	defer tempFile.Close()

	if _, err := tempFile.Write(sdp); err != nil {
		utils.Check(err)
	}

	ffplay := exec.Command("ffplay", "-window_title", content,
		"-protocol_whitelist", "pipe,udp,rtp,file", "-f", "sdp", "-i",
		tempFile.Name())

	ffplay.Stdout = os.Stdout
	ffplay.Stderr = os.Stderr

	err = ffplay.Start()
	utils.Check(err)

	err = ffplay.Wait()
	utils.Check(err)
}

func main() {

	neighbour := flag.String("neighbour", "", "address of the a network node neighbour")
	content := flag.String("content", "video.mp4", "specify what content to playback")

	flag.Parse()

	if *content == "video.mp4" {
		log.Printf("no content name specificed, defaulting to '%v'", *content)
	}

	if *neighbour == "" {
		log.Fatalf("neighbour address is mandatory to run the client")
	}

	_, err := netip.ParseAddrPort(*neighbour)
	utils.Check(err)

	// discovery phase - send discovery packet, get response, check if found or not

	clientUuid := uuid.New()
	log.Printf("created client id %v\n", clientUuid)

	conn := utils.SetupConnection("tcp", *neighbour)
	log.Printf("connected with neighbout '%v' via tcp\n", *neighbour)

	packet, err := packets.Discovery(clientUuid, *content).Encode()
	utils.Check(err)

	resultBytes := utils.SendAndWait(packet, conn)
	log.Printf("received response for discovery request, decoding...\n")

	result, err := packets.DecodePacket(resultBytes)
	utils.Check(err)

	if result.Header.Flags != packets.FOUND {
		log.Printf("content '%v' is not available in the network\n", *content)
		utils.CloseConnection(conn, *neighbour)
		return
	}

	log.Printf("content '%v' is available on the network, initiating stream request\n", *content)

	// stream phase - send stream request, wait for port to listen to

	packet, err = packets.Stream(clientUuid, *content).Encode()
	utils.Check(err)

	resultBytes = utils.SendAndWait(packet, conn)
	log.Printf("received response from stream request, decoding...\n")

	result, err = packets.DecodePacket(resultBytes)
	utils.Check(err)

	if result.Header.Flags != packets.PORT {
		log.Printf("something went wrong, did not receive PORT packet\n")
		utils.CloseConnection(conn, *neighbour)
		return
	}

	log.Printf("content '%v' is streaming at '%v'\n", *content, result.Payload.Port)

	sdpPort := strings.Split(result.Payload.Port, ":")[1]
	sdp := SetSdpPort(string(result.Payload.Sdp), sdpPort)
	log.Printf("received sdp file for the stream:\n%v", sdp)

	Listen([]byte(sdp), result.Payload.ContentName)
}
