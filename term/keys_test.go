package term

import (
	"reflect"
	"testing"
)

func keys(evs []Event) []Key {
	var out []Key
	for _, e := range evs {
		out = append(out, e.Key)
	}
	return out
}

func TestParseLegacy(t *testing.T) {
	evs, rest := Parse([]byte("ab\x1b[A\x1bOB\r\x7f\t\x03\x1b[15~\x1b[1;5C"))
	want := []Key{'a', 'b', KeyUp, KeyDown, KeyEnter, KeyBackspace, KeyTab, KeyCtrlC, KeyF5, KeyRight}
	if !reflect.DeepEqual(keys(evs), want) || rest != nil {
		t.Fatalf("got %v rest %q, want %v", keys(evs), rest, want)
	}
}

func TestParsePartial(t *testing.T) {
	evs, rest := Parse([]byte("x\x1b[1;"))
	if !reflect.DeepEqual(keys(evs), []Key{'x'}) || string(rest) != "\x1b[1;" {
		t.Fatalf("got %v rest %q", keys(evs), rest)
	}
	evs, rest = Parse(append(rest, '5', 'D'))
	if !reflect.DeepEqual(keys(evs), []Key{KeyLeft}) || rest != nil {
		t.Fatalf("after the rest: %v rest %q", keys(evs), rest)
	}
	// A lone ESC waits for more.
	if evs, rest := Parse([]byte{0x1b}); len(evs) != 0 || len(rest) != 1 {
		t.Fatalf("lone ESC: %v rest %q", evs, rest)
	}
	// ESC then a plain key is Esc, then the key.
	if evs, _ := Parse([]byte("\x1bq")); !reflect.DeepEqual(keys(evs), []Key{KeyEsc, 'q'}) {
		t.Fatalf("ESC q: %v", keys(evs))
	}
}
