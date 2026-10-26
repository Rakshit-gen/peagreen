package term

import (
	"io"
	"testing"
	"time"
)

func next(t *testing.T, ch <-chan Event) Event {
	t.Helper()
	select {
	case e := <-ch:
		return e
	case <-time.After(time.Second):
		t.Fatal("no event")
	}
	return Event{}
}

func TestReader(t *testing.T) {
	r, w := io.Pipe()
	ch := Read(r)
	w.Write([]byte("\x1b["))
	w.Write([]byte("B"))
	if e := next(t, ch); e.Key != KeyDown {
		t.Fatalf("split sequence: got %v", e.Key)
	}
	w.Write([]byte{0x1b})
	if e := next(t, ch); e.Key != KeyEsc {
		t.Fatalf("lone ESC: got %v", e.Key)
	}
	w.Write([]byte("z"))
	if e := next(t, ch); e.Key != 'z' {
		t.Fatalf("got %v", e.Key)
	}
	w.Close()
	if _, ok := <-ch; ok {
		t.Fatal("channel should close when input ends")
	}
}
