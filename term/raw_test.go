//go:build darwin || linux

package term

import (
	"os"
	"testing"
)

func TestNotATerminal(t *testing.T) {
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	defer r.Close()
	defer w.Close()
	if IsTerminal(int(r.Fd())) {
		t.Fatal("a pipe isn't a terminal")
	}
	if _, err := MakeRaw(int(r.Fd())); err == nil {
		t.Fatal("raw mode on a pipe should fail")
	}
}
