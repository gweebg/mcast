package utils

import (
	"encoding/json"
	"fmt"
	"log"
	"net"
	"os"
	"os/exec"
	"reflect"
	"strings"
	"sync"
	"time"
)

func PrintStruct(s interface{}) {

	sJson, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		log.Fatalf(err.Error())
	}
	fmt.Printf("%s\n", string(sJson))

}

func MustParseJson[T any](path string, validator ...func(T) bool) T {

	var result T

	data, err := os.ReadFile(path)
	if err != nil {
		log.Fatalf("file %s does not exists or is not acessible in the current path\n", path)
	}

	err = json.Unmarshal(data, &result)
	if err != nil {
		log.Fatalf("error while parsing json %s\n", err.Error())
	}

	switch len(validator) {

	case 1: // one validator function is passed
		if !validator[0](result) {
			log.Fatalf("json object is not valid for type %v\n", reflect.TypeOf(result))
		}
		break

	case 0: // no validator function is passed
		break

	default: // more than one validator function is passed
		log.Fatalf("only one validator function is accepted, but got %d\n", len(validator))

	}

	return result
}

func CloseConnection(conn net.Conn, addr string) {
	err := conn.Close()
	Check(err)
	log.Printf("(%v) closed connection\n", addr)
}

func Check(err error) {
	if err != nil {
		log.Fatalf(err.Error())
	}
}

func ReplacePortFromAddressString(address string, port string) string {
	split := strings.Split(address, ":")
	return split[0] + ":" + port
}

func SendAndWait(content []byte, conn net.Conn) []byte {

	_, err := conn.Write(content)
	Check(err)

	buffer := make([]byte, 1024)
	n, err := conn.Read(buffer)
	Check(err)

	return buffer[:n]
}

func SetupConnection(network string, address string) net.Conn {

	conn, err := net.Dial(network, address)
	Check(err)

	return conn
}

func ListenStream(address string) {

	TsMtu := 188

	addr, err := net.ResolveUDPAddr("udp", address)
	Check(err)

	udpConn, err := net.ListenUDP("udp", addr)
	Check(err)

	defer func(udpConn *net.UDPConn) {
		Check(udpConn.Close())
	}(udpConn)

	// create ffplay process
	ffplay := exec.Command("ffplay", "-f", "mpegts", "-")

	ffplay.Stdout = os.Stdout
	ffplay.Stderr = os.Stderr

	stdin, err := ffplay.StdinPipe()
	Check(err)

	// start ffplay process
	err = ffplay.Start()
	Check(err)

	// goroutine for continuously reading and writing MPEG TS packets to ffplay
	go func() {

		buffer := make([]byte, TsMtu*10) // 188 bytes per MPEG TS packet

		for {

			n, _, err := udpConn.ReadFromUDP(buffer)
			if err != nil {
				log.Printf("stop receiving stream from '%v', exitting\n", address)
				break
			}

			_, err = stdin.Write(buffer[:n])
			Check(err)
		}
	}()

	time.Sleep(100 * time.Millisecond)

	err = ffplay.Wait()
	Check(err)
}

type SdpDatabase struct {
	Files map[string][]byte
	Mu    sync.RWMutex
}

func NewSdpDatabase() *SdpDatabase {
	return &SdpDatabase{
		Files: make(map[string][]byte),
	}
}

func (s *SdpDatabase) SetSdp(contentName string, content []byte) {

	s.Mu.Lock()
	defer s.Mu.Unlock()

	s.Files[contentName] = content
}

func (s *SdpDatabase) MustGetSdp(contentName string) []byte {

	s.Mu.RLock()
	defer s.Mu.RUnlock()

	sdp, exists := s.Files[contentName]
	if !exists {
		log.Fatalf("(sdp database) sdp file for '%v' should exist, but it does not.\n", contentName)
	}

	return sdp
}
