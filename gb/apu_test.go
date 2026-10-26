package gb

import "testing"

// TestSquareTone plays channel 2 at 131072/(2048-1917) = 1000.5 Hz for a
// second and counts the times the output crosses zero going up.
func TestSquareTone(t *testing.T) {
	m, err := New(blankROM(0, 0, 0))
	if err != nil {
		t.Fatal(err)
	}
	m.EnableAudio(48000)
	for _, w := range [][2]uint16{
		{0xFF26, 0x80}, {0xFF24, 0x77}, {0xFF25, 0x02},
		{0xFF16, 0x80}, {0xFF17, 0xF0}, {0xFF18, 1917 & 0xFF}, {0xFF19, 0x80 | 1917>>8},
	} {
		m.Write(w[0], uint8(w[1]))
	}
	var left, right []float32
	for m.Cycles < cyclesPerSecond {
		m.tick()
	}
	s := m.Audio()
	for i := 0; i < len(s); i += 2 {
		left, right = append(left, s[i]), append(right, s[i+1])
	}
	if n := len(left); n < 47990 || n > 48010 {
		t.Fatalf("%d samples in a second, want 48000", n)
	}
	ups := 0
	for i := 1; i < len(right); i++ {
		if right[i-1] < 0 && right[i] >= 0 {
			ups++
		}
	}
	if ups < 995 || ups > 1005 {
		t.Fatalf("%d cycles in a second, want about 1000", ups)
	}
	for _, v := range left {
		if v != 0 {
			t.Fatal("NR51 0x02 should keep channel 2 out of the left side")
		}
	}
}
