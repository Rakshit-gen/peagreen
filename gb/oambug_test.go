package gb

import "testing"

// scanning puts the PPU in mode 2 on line 10, at the M-cycle that reads row.
func scanning(row int) *Machine {
	m, _ := New(blankROM(0, 0, 0))
	p := &m.PPU
	p.LCDC, p.LY, p.Dot = 0x80, 10, row*4
	p.setMode(modeOAM)
	for i := range p.OAM {
		p.OAM[i] = uint8(i)
	}
	return m
}

func TestOAMBugWrite(t *testing.T) {
	m := scanning(2)
	p := &m.PPU
	a, b, c := p.oamWord(2, 0), p.oamWord(1, 0), p.oamWord(1, 2)
	m.oamBug(0xFE40, oamWrite)
	if got, want := p.oamWord(2, 0), (a^c)&(b^c)^c; got != want {
		t.Fatalf("first word %04x, want %04x", got, want)
	}
	for i := 1; i < 4; i++ {
		if p.oamWord(2, i) != p.oamWord(1, i) {
			t.Fatalf("word %d wasn't copied from the row before", i)
		}
	}
	if p.OAM[0] != 0 || p.OAM[24] != 24 {
		t.Fatal("other rows changed")
	}
}

func TestOAMBugOnlyInMode2(t *testing.T) {
	m := scanning(5)
	m.PPU.setMode(modeHBlank)
	m.oamBug(0xFE00, oamRead)
	m.PPU.setMode(modeOAM)
	m.oamBug(0xC000, oamRead)
	scanning(0).oamBug(0xFE00, oamWrite) // the first row is never hit
	for i, v := range m.PPU.OAM {
		if v != uint8(i) {
			t.Fatalf("OAM changed at %d", i)
		}
	}
}
