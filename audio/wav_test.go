package audio

import (
	"encoding/binary"
	"os"
	"path/filepath"
	"testing"
)

func TestWAV(t *testing.T) {
	path := filepath.Join(t.TempDir(), "out.wav")
	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	w, err := NewWAV(f, 48000)
	if err != nil {
		t.Fatal(err)
	}
	w.Write([]float32{0, 1, -1, 2}) // 2 clips to 1
	w.Write([]float32{0.5, -0.5})
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	f.Close()
	b, _ := os.ReadFile(path)
	if len(b) != 44+12 {
		t.Fatalf("%d bytes, want 56", len(b))
	}
	if got := binary.LittleEndian.Uint32(b[40:]); got != 12 {
		t.Fatalf("data size %d, want 12", got)
	}
	if got := binary.LittleEndian.Uint32(b[4:]); got != 48 {
		t.Fatalf("riff size %d, want 48", got)
	}
	want := []int16{0, 32767, -32767, 32767, 16384, -16384}
	for i, w := range want {
		if got := int16(binary.LittleEndian.Uint16(b[44+i*2:])); got != w {
			t.Errorf("sample %d = %d, want %d", i, got, w)
		}
	}
}
