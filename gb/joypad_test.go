package gb

import "testing"

func TestJoypad(t *testing.T) {
	m, err := New(blankROM(0, 0, 0))
	if err != nil {
		t.Fatal(err)
	}
	m.IF = 0
	m.Press(Start | Left)
	if m.IF&IntJoypad == 0 {
		t.Fatal("pressing should raise the joypad interrupt")
	}
	m.Write(0xFF00, 0x20) // d-pad row
	if got := m.Read(0xFF00); got != 0xED {
		t.Fatalf("d-pad row = %#x, want 0xED (left held)", got)
	}
	m.Write(0xFF00, 0x10) // button row
	if got := m.Read(0xFF00); got != 0xD7 {
		t.Fatalf("button row = %#x, want 0xD7 (start held)", got)
	}
	m.Release(Start)
	if got := m.Read(0xFF00); got != 0xDF {
		t.Fatalf("after release = %#x, want 0xDF", got)
	}
	m.IF = 0
	m.Press(Left)
	if m.IF != 0 {
		t.Fatal("holding a held button again shouldn't interrupt")
	}
}
