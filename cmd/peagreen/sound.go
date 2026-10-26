package main

import (
	"encoding/binary"
	"errors"
	"io"
	"math"
	"os/exec"
)

// peagreen has no sound driver of its own (the standard library has none
// to offer), so it plays sound through whichever of these is installed,
// feeding it raw 32-bit float stereo at soundRate.
const soundRate = 48000

var soundPlayers = [][]string{
	{"ffplay", "-nodisp", "-autoexit", "-loglevel", "error", "-fflags", "nobuffer",
		"-f", "f32le", "-ar", "48000", "-ch_layout", "stereo", "-i", "-"},
	{"play", "-q", "-t", "f32", "-r", "48000", "-c", "2", "-"},               // sox
	{"pacat", "--raw", "--format=float32le", "--rate=48000", "--channels=2"}, // PulseAudio, PipeWire
	{"aplay", "-q", "-t", "raw", "-f", "FLOAT_LE", "-r", "48000", "-c", "2"}, // ALSA
}

// speaker sends samples to a player program without ever making the game
// wait: if the player falls behind, samples are dropped.
type speaker struct {
	cmd    *exec.Cmd
	in     io.WriteCloser
	frames chan []byte
	done   chan struct{}
}

func openSpeaker() (*speaker, error) {
	for _, args := range soundPlayers {
		if _, err := exec.LookPath(args[0]); err != nil {
			continue
		}
		cmd := exec.Command(args[0], args[1:]...)
		in, err := cmd.StdinPipe()
		if err != nil {
			return nil, err
		}
		if err := cmd.Start(); err != nil {
			return nil, err
		}
		s := &speaker{cmd: cmd, in: in, frames: make(chan []byte, 8), done: make(chan struct{})}
		go s.feed()
		return s, nil
	}
	return nil, errors.New("no sound: install ffplay (ffmpeg) or sox, or play with -sound=false")
}

func (s *speaker) feed() {
	defer close(s.done)
	for b := range s.frames {
		if _, err := s.in.Write(b); err != nil {
			for range s.frames {
			}
			return
		}
	}
}

// play queues one frame of samples.
func (s *speaker) play(samples []float32) {
	b := make([]byte, 4*len(samples))
	for i, v := range samples {
		binary.LittleEndian.PutUint32(b[i*4:], math.Float32bits(v))
	}
	select {
	case s.frames <- b:
	default:
	}
}

func (s *speaker) close() {
	close(s.frames)
	<-s.done
	s.in.Close()
	s.cmd.Process.Kill()
	s.cmd.Wait()
}
