package gb

// Screen size in pixels.
const (
	Width  = 160
	Height = 144
)

// PPU modes, in STAT bits 0-1.
const (
	modeHBlank = 0
	modeVBlank = 1
	modeOAM    = 2
	modeDraw   = 3
)

// PPU is the picture processor. It draws one line at a time, when the line
// enters mode 3, which is close enough for anything that doesn't change
// registers in the middle of a line.
type PPU struct {
	VRAM [0x2000]byte
	OAM  [0xA0]byte

	LCDC, STAT, SCY, SCX, LY, LYC uint8
	BGP, OBP0, OBP1, WY, WX       uint8
	DMA                           uint8

	Dot      int  // dot (T-cycle) within the current line
	WinLine  int  // the window's own line counter
	WinY     bool // LY has matched WY this frame, so the window may show
	StatLine bool // the STAT interrupt line, which fires on its rising edge
	DrawEnd  int  // the dot this line's mode 3 ends on
	ModeAt   int  // the dot the mode last changed on

	// Screen holds shades 0 (lightest) to 3 for the frame being drawn. It's
	// complete between the start of vblank and the start of the next frame.
	Screen [Height][Width]uint8
	Frames uint64
}

const lineDots = 456

func (p *PPU) mode() uint8 { return p.STAT & 3 }

func (p *PPU) setMode(md uint8) { p.STAT = p.STAT&^3 | md }

func (m *Machine) ppuTick() {
	p := &m.PPU
	if p.LCDC&0x80 == 0 {
		return
	}
	p.Dot += 4
	if p.LY < Height {
		switch {
		case p.Dot == 80:
			p.setMode(modeDraw)
			p.ModeAt, p.DrawEnd = p.Dot, 80+p.drawLength()
			m.renderLine()
		case p.mode() == modeDraw && p.Dot >= p.DrawEnd:
			p.setMode(modeHBlank)
			p.ModeAt = p.Dot
		}
	}
	if p.Dot >= lineDots {
		p.Dot -= lineDots
		m.nextLine()
	}
	m.updateStat()
}

func (m *Machine) nextLine() {
	p := &m.PPU
	p.LY++
	switch {
	case p.LY == 154:
		p.LY = 0
		p.WinLine, p.WinY = 0, false
		p.setMode(modeOAM)
	case p.LY == Height:
		p.setMode(modeVBlank)
		m.IF |= IntVBlank
		p.Frames++
	case p.LY < Height:
		p.setMode(modeOAM)
	}
	if p.LY == p.WY {
		p.WinY = true
	}
}

// drawLength is how many dots mode 3 takes on this line. It's 172 at the
// least; the fine scroll throws pixels away first, the window restarts the
// fetcher, and each sprite stalls it, by 6 dots and up to 5 more depending
// on where the sprite sits against the background tiles. (The rules are
// from the Pan Docs' "Mode 3 length".)
func (p *PPU) drawLength() int {
	n := 172 + int(p.SCX&7)
	if p.LCDC&0x20 != 0 && p.WinY && p.WX <= 166 {
		n += 6
	}
	if p.LCDC&2 == 0 {
		return n
	}
	var line [10]int
	var seen [32]bool // background tiles a sprite has already waited on
	stall := 0
	for _, i := range p.lineSprites(&line) {
		x := int(p.OAM[i*4+1])
		if x >= 168 {
			continue
		}
		// Where the sprite's left edge is, counted from the left of the
		// tile just off screen, so it's never negative.
		at := x + int(p.SCX&7)
		if tile := at / 8; !seen[tile] {
			seen[tile] = true
			stall += max(0, 5-at%8)
		}
		stall += 6
	}
	// The CPU sees mode 0 three dots sooner after sprites than these add
	// up to, going by mooneye's intr_2_mode0_timing_sprites, which checks
	// 106 layouts against the hardware.
	return n + max(0, stall-3)
}

// updateStat raises the STAT interrupt when any enabled source turns on
// while none was on before.
func (m *Machine) updateStat() {
	p := &m.PPU
	if p.LY == p.LYC {
		p.STAT |= 4
	} else {
		p.STAT &^= 4
	}
	md := p.mode()
	line := p.STAT&0x40 != 0 && p.LY == p.LYC ||
		p.STAT&0x20 != 0 && md == modeOAM ||
		p.STAT&0x10 != 0 && md == modeVBlank ||
		p.STAT&0x08 != 0 && md == modeHBlank
	if p.LCDC&0x80 == 0 {
		line = false
	}
	if line && !p.StatLine {
		m.IF |= IntSTAT
	}
	p.StatLine = line
}

// statRead is STAT as the CPU sees it. The mode bits change one M-cycle
// after the PPU switches mode: the switch to mode 3 or 0 raises its
// interrupt at once, but a read in the same M-cycle still shows the old
// mode.
func (p *PPU) statRead() uint8 {
	if p.LCDC&0x80 != 0 && p.LY < Height && p.Dot == p.ModeAt {
		switch p.mode() {
		case modeDraw:
			return p.STAT&^3 | modeOAM
		case modeHBlank:
			return p.STAT&^3 | modeDraw
		}
	}
	return p.STAT
}

func (m *Machine) ppuRead(a uint16) uint8 {
	p := &m.PPU
	switch a {
	case 0xFF40:
		return p.LCDC
	case 0xFF41:
		return p.statRead() | 0x80
	case 0xFF42:
		return p.SCY
	case 0xFF43:
		return p.SCX
	case 0xFF44:
		return p.LY
	case 0xFF45:
		return p.LYC
	case 0xFF46:
		return p.DMA
	case 0xFF47:
		return p.BGP
	case 0xFF48:
		return p.OBP0
	case 0xFF49:
		return p.OBP1
	case 0xFF4A:
		return p.WY
	case 0xFF4B:
		return p.WX
	}
	return 0xFF
}

func (m *Machine) ppuWrite(a uint16, v uint8) {
	p := &m.PPU
	switch a {
	case 0xFF40:
		m.setLCDC(v)
	case 0xFF41:
		p.STAT = p.STAT&7 | v&0x78
	case 0xFF42:
		p.SCY = v
	case 0xFF43:
		p.SCX = v
	case 0xFF45:
		p.LYC = v
	case 0xFF46:
		m.dma(v)
	case 0xFF47:
		p.BGP = v
	case 0xFF48:
		p.OBP0 = v
	case 0xFF49:
		p.OBP1 = v
	case 0xFF4A:
		p.WY = v
	case 0xFF4B:
		p.WX = v
	}
	m.updateStat()
}

// setLCDC turns the LCD off and on with bit 7. Off, LY sits at 0 and the
// PPU stays in mode 0; back on, it starts again at the top of a frame.
func (m *Machine) setLCDC(v uint8) {
	p := &m.PPU
	was := p.LCDC&0x80 != 0
	p.LCDC = v
	switch on := v&0x80 != 0; {
	case was && !on:
		p.LY, p.Dot = 0, 0
		p.setMode(modeHBlank)
	case !was && on:
		p.LY, p.Dot = 0, 0
		p.WinLine, p.WinY = 0, p.WY == 0
		p.setMode(modeOAM)
	}
}

// DMAState is an OAM DMA transfer: a cycle of setup after the write to
// 0xFF46, then one byte per M-cycle for 160 M-cycles. A new write restarts
// it, and the old transfer keeps going through the new one's setup cycle.
type DMAState struct {
	Active  bool
	Src     uint16
	Pos     int
	Pending bool // a transfer starts next M-cycle
	Next    uint16
}

func (m *Machine) dma(v uint8) {
	m.PPU.DMA = v
	src := uint16(v) << 8
	if v >= 0xE0 {
		src -= 0x2000 // 0xE000 and up reads WRAM
	}
	m.DMAState.Pending, m.DMAState.Next = true, src
}

func (m *Machine) dmaTick() {
	d := &m.DMAState
	if d.Active {
		if d.Pos == len(m.PPU.OAM) {
			d.Active = false
		} else {
			m.PPU.OAM[d.Pos] = m.Read(d.Src + uint16(d.Pos))
			d.Pos++
		}
	}
	if d.Pending {
		d.Pending = false
		d.Active, d.Src, d.Pos = true, d.Next, 0
	}
}

// blocked reports whether the CPU is shut out of address a, by a running
// DMA or by the PPU.
func (m *Machine) blocked(a uint16) bool {
	if a >= 0xFE00 && a < 0xFF00 && m.DMAState.Active {
		return true
	}
	// The PPU has OAM to itself while it scans it (mode 2) and while it
	// draws (mode 3), and VRAM while it draws. The CPU reads 0xFF and its
	// writes are lost.
	switch m.PPU.statRead() & 3 {
	case modeOAM:
		return a >= 0xFE00 && a < 0xFEA0
	case modeDraw:
		return a >= 0x8000 && a < 0xA000 || a >= 0xFE00 && a < 0xFEA0
	}
	return false
}
