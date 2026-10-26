package gb

import "testing"

// onLine puts an LCD-on PPU at the start of line ly with sprites shown.
func onLine(ly uint8) *PPU {
	p := &PPU{LCDC: 0x83, LY: ly}
	for i := range 40 {
		p.OAM[i*4] = 0 // all off screen
	}
	return p
}

func TestDrawLength(t *testing.T) {
	sprite := func(p *PPU, i int, x uint8) {
		p.OAM[i*4], p.OAM[i*4+1] = 16+p.LY, x
	}
	for _, c := range []struct {
		name string
		set  func(p *PPU)
		want int
	}{
		{"plain", func(p *PPU) {}, 172},
		{"fine scroll", func(p *PPU) { p.SCX = 0x23 }, 175},
		{"window", func(p *PPU) { p.LCDC |= 0x20; p.WinY = true; p.WX = 7 }, 178},
		{"sprite at x 0", func(p *PPU) { sprite(p, 0, 0) }, 172 + 11 - 3},
		{"two sprites in one tile", func(p *PPU) { sprite(p, 0, 0); sprite(p, 1, 0) }, 172 + 17 - 3},
		{"sprite past the right edge", func(p *PPU) { sprite(p, 0, 168) }, 172},
		{"sprites hidden", func(p *PPU) { sprite(p, 0, 8); p.LCDC &^= 2 }, 172},
	} {
		p := onLine(10)
		c.set(p)
		if got := p.drawLength(); got != c.want {
			t.Errorf("%s: mode 3 takes %d dots, want %d", c.name, got, c.want)
		}
	}
}

func TestStatLagsModeChange(t *testing.T) {
	m, _ := New(blankROM(0, 0, 0))
	p := &m.PPU
	p.LCDC, p.LY, p.Dot = 0x80, 5, 76
	p.setMode(modeOAM)
	m.ppuTick() // dot 80: drawing starts
	if p.mode() != modeDraw || p.statRead()&3 != modeOAM {
		t.Fatalf("in the M-cycle mode 3 starts, STAT should still say 2, says %d", p.statRead()&3)
	}
	m.ppuTick()
	if p.statRead()&3 != modeDraw {
		t.Fatal("STAT should say 3 an M-cycle later")
	}
}

func TestLYCFrozenWhileOff(t *testing.T) {
	m, _ := New(blankROM(0, 0, 0))
	p := &m.PPU
	p.LY, p.LYC = 0, 0
	m.updateStat()
	m.setLCDC(0)
	p.LYC = 9
	m.updateStat()
	if p.STAT&4 == 0 {
		t.Fatal("the LY=LYC bit should keep its value while the LCD is off")
	}
}

func TestAccessAsMode3Starts(t *testing.T) {
	m, _ := New(blankROM(0, 0, 0))
	p := &m.PPU
	p.LCDC, p.LY, p.Dot = 0x80, 5, 76
	p.setMode(modeOAM)
	m.ppuTick()
	if !m.blocked(0x8000, false) {
		t.Error("VRAM reads should close as the PPU enters mode 3")
	}
	if m.blocked(0x8000, true) || m.blocked(0xFE00, true) {
		t.Error("writes should still land in the M-cycle mode 3 starts")
	}
	m.ppuTick()
	if !m.blocked(0x8000, true) || !m.blocked(0xFE00, true) {
		t.Error("writes should be lost once STAT shows mode 3")
	}
	if m.blocked(0xC000, false) || m.blocked(0xC000, true) {
		t.Error("WRAM should never be blocked by the PPU")
	}
}

func TestDMAShutsOAMOnFirstByte(t *testing.T) {
	m, _ := New(blankROM(0, 0, 0))
	m.setLCDC(0)
	m.dma(0xC0)
	m.dmaTick() // set up
	if m.blocked(0xFE00, false) {
		t.Fatal("OAM should stay open until a byte is copied")
	}
	m.dmaTick()
	if !m.blocked(0xFE00, false) || !m.blocked(0xFE9F, true) {
		t.Fatal("OAM should be shut while the transfer copies")
	}
	m.dma(0xC0)
	m.dmaTick()
	if !m.blocked(0xFE00, false) {
		t.Fatal("a restart should keep OAM shut")
	}
}
