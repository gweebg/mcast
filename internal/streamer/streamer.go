package streamer

import (
	"github.com/gweebg/mcast/internal/utils"
	"log"
	"net"
	"os"
	"os/exec"
	"strings"
)

const (
	VideoDir string = "resources/videos/"
	TsMtu    int    = 188
)

type Option func(*Streamer)

type Streamer struct {
	Address string
	conn    net.Conn

	SdpFile     []byte
	contentPath string
	ContentName string

	IsStreaming bool
	StreamProc  *exec.Cmd
}

func New(options ...Option) *Streamer {

	streamer := &Streamer{
		Address:     "127.0.0.1:5000",
		ContentName: "video.mp4",
		IsStreaming: false,
	}

	for _, opt := range options {
		opt(streamer)
	}

	conn, err := net.Dial("udp", streamer.Address)
	utils.Check(err)

	streamer.conn = conn
	streamer.contentPath = VideoDir + streamer.ContentName

	return streamer

}

func WithAddress(addr string) Option {
	return func(s *Streamer) {
		s.Address = addr
	}
}

func WithContentName(name string) Option {
	return func(s *Streamer) {
		s.ContentName = name
	}
}

// Stream starts the streaming process of the video.
// Firstly sets up the mpeg-ts resources, then streams the content via udp to via conn.
func (s *Streamer) Stream() {

	s.IsStreaming = true

	ffmpeg := exec.Command("ffmpeg", "-re", "-ss", "00:00:00.000",
		"-stream_loop", "-1", "-i", s.contentPath, "-vcodec",
		"copy", "-an", "-f", "rtp", "rtp://"+s.Address,
	)

	s.StreamProc = ffmpeg

	log.Printf("(handling %v) started streaming '%v'\n", s.Address, s.ContentName)

	err := ffmpeg.Run()
	if err != nil {
		log.Fatalf("(streamer) could not initiate ffmpeg streaming process for '%v'\n", s.ContentName)
	}
}

func (s *Streamer) GetSdp() ([]byte, error) {

	sdpFile := strings.Split(s.contentPath, ".")[0] + ".sdp"
	sdp, err := os.ReadFile(sdpFile)

	return sdp, err

}

// Teardown stops the streaming by triggering the stopChannel.
func (s *Streamer) Teardown() {
	s.IsStreaming = false
	s.StreamProc.Process.Kill()
}
