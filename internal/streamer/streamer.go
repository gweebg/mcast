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
	TsDir    string = "resources/ts/"

	TsMtu int = 188
)

type Option func(*Streamer)

type Streamer struct {
	Address     string
	ContentName string
	IsStreaming bool

	contentPath         string
	transportStreamPath string
	stopChannel         chan struct{}

	conn net.Conn
}

func New(options ...Option) *Streamer {

	streamer := &Streamer{
		Address:     "127.0.0.1:5000",
		ContentName: "video.mp4",
		IsStreaming: false,
		stopChannel: make(chan struct{}),
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

// encodeTransportStream, encodes the video (ContentName) into an MPEG Transport Stream using FFMPEG.
func (s *Streamer) encodeTransportStream() (err error) {

	log.Printf("encoding video '%v' into transport stream...\n", s.ContentName)
	ffmpeg := exec.Command("ffmpeg",
		"-i", s.contentPath,
		"-c:v", "mpeg2video", "-c:a", "mp2",
		s.transportStreamPath,
	)

	err = ffmpeg.Run()
	return err
}

// Returns the transport stream file descriptor.
func (s *Streamer) getTransportStream() *os.File {

	file, err := os.Open(s.transportStreamPath)
	utils.Check(err)

	return file
}

// SetupVideo sets up the necessary resources to start the video transmission.
// Returns the file descriptor of the transport stream.
func (s *Streamer) SetupVideo() *os.File {

	tsExists := false

	transportStreamFilename := strings.Split(s.ContentName, ".")[0] + ".ts"
	s.transportStreamPath = TsDir + transportStreamFilename

	entries, err := os.ReadDir(TsDir)
	utils.Check(err)

	for _, e := range entries {
		if e.Name() == transportStreamFilename {
			tsExists = true
		}
	}

	if !tsExists {
		err = s.encodeTransportStream()

		if err != nil {
			log.Fatalf("could not encode '%v' into mpeg-ts\n%v", s.ContentName, err.Error())
		}

		log.Printf("finished encoding '%v'\n", s.ContentName)
	}

	return s.getTransportStream()
}

// Cleans up the dangling connection, command and streaming status once the streamer receives the stop signal.
func (s *Streamer) cleanup(cmd *exec.Cmd, close bool) {

	s.IsStreaming = false // no longer streaming

	err := cmd.Process.Kill() // kill the ffplay sync command
	utils.Check(err)

	if close {
		err = s.conn.Close() // closing udp connection
		utils.Check(err)
	}
}

// Stream starts the streaming process of the video.
// Firstly sets up the mpeg-ts resources, then streams the content via udp to via conn.
func (s *Streamer) Stream() {

	s.IsStreaming = true // todo: not updating ?

	ts := s.SetupVideo()

	defer func(ts *os.File) {
		err := ts.Close()
		if err != nil {
			utils.Check(err)
		}
	}(ts)

	// used to sync the mpeg-ts packets with realtime (1s realtime matching with 1s video)
	sync := exec.Command("ffplay", "-nodisp", "-af", "volume=0.0", "-")

	syncIn, err := sync.StdinPipe()
	utils.Check(err)

	err = sync.Start() // start sync process via ffplay
	utils.Check(err)

	seq := 0
	buffer := make([]byte, TsMtu*10)
	for {

		select {

		case <-s.stopChannel: // breakdown connection and stop streaming
			s.cleanup(sync, true)
			break

		default: // transmit the data via udp
			size, err := ts.Read(buffer)
			if err != nil {
				_, err := ts.Seek(0, 0) // error on read => loop
				utils.Check(err)
				seq = 0 // set sequence to 0
			} // loop over the video once we reach the end

			_, err = syncIn.Write(buffer)
			if err != nil {
				log.Println("broken pipe on syncIn")
			}

			size, err = s.conn.Write(buffer[:size])
			if err != nil {
				log.Printf("lost connection with '%v'\n", s.Address)
				s.cleanup(sync, false)
				return
			}

			// log.Printf("sent packet #%d (%d bytes)\n", seq, size)
			seq++
		}
	}
}

// Teardown stops the streaming by triggering the stopChannel.
func (s *Streamer) Teardown() {
	log.Println("teardown triggered")
	s.stopChannel <- struct{}{}
}
