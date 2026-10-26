package gb

import (
	"bytes"
	"testing"
)

func TestStateRoundTrip(t *testing.T) {
	m := loadROM(t, "dmg-acid2.gb")
	for range 10 {
		m.RunFrame()
	}
	var buf bytes.Buffer
	if err := WriteState(&buf, m.Save()); err != nil {
		t.Fatal(err)
	}
	for range 20 {
		m.RunFrame()
	}
	want, cycles := m.PPU.Screen, m.Cycles

	s, err := ReadState(&buf)
	if err != nil {
		t.Fatal(err)
	}
	if err := m.Load(s); err != nil {
		t.Fatal(err)
	}
	for range 20 {
		m.RunFrame()
	}
	if m.PPU.Screen != want || m.Cycles != cycles {
		t.Fatalf("after loading, 20 frames ended at cycle %d, want %d (same screen: %v)", m.Cycles, cycles, m.PPU.Screen == want)
	}
}

func TestStateOtherGame(t *testing.T) {
	a, _ := New(blankROM(0x00, 0, 0))
	rom := blankROM(0x00, 0, 0)
	copy(rom[0x134:], "OTHER")
	fixChecksums(rom)
	b, _ := New(rom)
	if err := b.Load(a.Save()); err == nil {
		t.Fatal("loaded a state from another game")
	}
}

func TestStateKeepsCartRAM(t *testing.T) {
	m, err := New(blankROM(0x03, 0, 2))
	if err != nil {
		t.Fatal(err)
	}
	m.Cart.RAM[5], m.Cart.ROMBank = 7, 1
	s := m.Save()
	m.Cart.RAM[5] = 9
	if s.Cart.RAM[5] != 7 {
		t.Fatal("the snapshot shares RAM with the machine")
	}
	ram := m.Cart.RAM
	if err := m.Load(s); err != nil {
		t.Fatal(err)
	}
	if m.Cart.RAM[5] != 7 || &m.Cart.RAM[0] != &ram[0] {
		t.Fatal("loading should copy the RAM back into the cartridge's own")
	}
}
