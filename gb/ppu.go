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

	Dot      int  // dot (T-cycle) within the current line
	StatLine bool // the STAT interrupt line, which fires on its rising edge

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
		switch p.Dot {
		case 80:
			p.setMode(modeDraw)
			m.renderLine()
		case 80 + 172:
			p.setMode(modeHBlank)
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
		p.setMode(modeOAM)
	case p.LY == Height:
		p.setMode(modeVBlank)
		m.IF |= IntVBlank
		p.Frames++
	case p.LY < Height:
		p.setMode(modeOAM)
	}
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

func (m *Machine) ppuRead(a uint16) uint8 {
	p := &m.PPU
	switch a {
	case 0xFF40:
		return p.LCDC
	case 0xFF41:
		return p.STAT | 0x80
	case 0xFF42:
		return p.SCY
	case 0xFF43:
		return p.SCX
	case 0xFF44:
		return p.LY
	case 0xFF45:
		return p.LYC
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
		p.setMode(modeOAM)
	}
}

// renderLine draws line LY into Screen.
func (m *Machine) renderLine() {}
