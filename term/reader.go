package term

import (
	"io"
	"time"
)

// escWait is how long a lone ESC waits for the rest of a sequence before
// it counts as the Esc key.
const escWait = 30 * time.Millisecond

// Read reads key events from r and sends them on the channel, which is
// closed when r fails or ends.
func Read(r io.Reader) <-chan Event {
	chunks := make(chan []byte)
	go func() {
		defer close(chunks)
		buf := make([]byte, 256)
		for {
			n, err := r.Read(buf)
			if n > 0 {
				chunks <- append([]byte(nil), buf[:n]...)
			}
			if err != nil {
				return
			}
		}
	}()
	out := make(chan Event, 64)
	go func() {
		defer close(out)
		var rest []byte
		send := func(evs []Event) {
			for _, e := range evs {
				out <- e
			}
		}
		for {
			var chunk []byte
			ok := true
			if len(rest) > 0 {
				select {
				case chunk, ok = <-chunks:
				case <-time.After(escWait):
					// Nothing followed: rest starts with the Esc key.
					out <- Event{Key: KeyEsc}
					var evs []Event
					evs, rest = Parse(rest[1:])
					send(evs)
					continue
				}
			} else {
				chunk, ok = <-chunks
			}
			if !ok {
				return
			}
			var evs []Event
			evs, rest = Parse(append(rest, chunk...))
			rest = append([]byte(nil), rest...)
			send(evs)
		}
	}()
	return out
}
