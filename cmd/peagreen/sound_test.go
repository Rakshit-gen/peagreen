package main

import (
	"encoding/binary"
	"io"
	"math"
	"testing"
	"time"
)

func TestSpeakerDropsWhenBehind(t *testing.T) {
	r, w := io.Pipe()
	s := &speaker{in: w, frames: make(chan []byte, 8), done: make(chan struct{})}
	go s.feed()

	ok := make(chan bool)
	go func() {
		for range 100 { // the reader is stalled, so these can't all fit
			s.play([]float32{0.5, -0.25})
		}
		ok <- true
	}()
	select {
	case <-ok:
	case <-time.After(2 * time.Second):
		t.Fatal("play blocked on a stalled player")
	}

	b := make([]byte, 8)
	if _, err := io.ReadFull(r, b); err != nil {
		t.Fatal(err)
	}
	for i, want := range []float32{0.5, -0.25} {
		if got := math.Float32frombits(binary.LittleEndian.Uint32(b[i*4:])); got != want {
			t.Errorf("sample %d = %v, want %v", i, got, want)
		}
	}
	r.Close()
	close(s.frames)
	<-s.done
}
