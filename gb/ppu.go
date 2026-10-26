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

	Dot      int   // dot (T-cycle) within the current line
	WinLine  int   // the window's own line counter
	WinY     bool  // LY has matched WY this frame, so the window may show
	StatLine bool  // the STAT interrupt line, which fires on its rising edge
	DrawEnd  int   // the dot this line's mode 3 ends on
	ModeAt   int   // when the mode last changed, as LY*456+Dot
	PrevMode uint8 // the mode before that
	Fresh    bool  // this is the first line since the LCD was turned on

	// Screen holds shades 0 (lightest) to 3 for the frame being drawn. It's
	// complete between the start of vblank and the start of the next frame.
	Screen [Height][Width]uint8
	Frames uint64
}

const lineDots = 456

func (p *PPU) mode() uint8 { return p.STAT & 3 }

func (p *PPU) setMode(md uint8) {
	p.PrevMode, p.ModeAt = p.mode(), p.now()
	p.STAT = p.STAT&^3 | md
}

// now is the dot within the frame.
func (p *PPU) now() int { return int(p.LY)*lineDots + p.Dot }

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
			p.DrawEnd = 80 + p.drawLength()
			m.renderLine()
		case p.mode() == modeDraw && p.Dot >= p.DrawEnd:
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
	p.Fresh = false
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
	// With the LCD off the comparison clock stops: the LY=LYC bit keeps
	// its last value, and so does the interrupt line, so turning the LCD
	// back on only interrupts if the line then rises.
	if p.LCDC&0x80 == 0 {
		return
	}
	if p.LY == p.LYC {
		p.STAT |= 4
	} else {
		p.STAT &^= 4
	}
	md := p.mode()
	line := p.STAT&0x44 == 0x44 ||
		p.STAT&0x20 != 0 && md == modeOAM ||
		// The OAM source fires once more as vblank starts, as if line
		// 144 began with a mode 2 like the others.
		p.STAT&0x20 != 0 && p.LY == Height && p.Dot < 4 ||
		p.STAT&0x10 != 0 && md == modeVBlank ||
		p.STAT&0x08 != 0 && md == modeHBlank
	if line && !p.StatLine {
		m.IF |= IntSTAT
	}
	p.StatLine = line
}

// statRead is STAT as the CPU sees it. The mode bits change one M-cycle
// after the PPU switches mode: the switch raises its interrupt at once,
// but a read in the same M-cycle still shows the old mode. And in the
// M-cycle LY changes, the LY=LYC bit reads 0 while the comparison settles.
func (p *PPU) statRead() uint8 {
	stat := p.STAT
	if p.LCDC&0x80 == 0 {
		return stat
	}
	if p.now() == p.ModeAt {
		stat = stat&^3 | p.PrevMode
	}
	if p.Dot == 0 {
		stat &^= 4
	}
	return stat
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
		// The first line after turning on skips the OAM scan: STAT says
		// mode 0 until drawing starts. It also starts a little way in,
		// so it ends one M-cycle early.
		p.LY, p.Dot, p.Fresh = 0, 4, true
		p.WinLine, p.WinY = 0, p.WY == 0
		p.setMode(modeHBlank)
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
	Hold    bool // a transfer copied a byte this M-cycle, so OAM is shut
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
	// OAM shuts with the first byte copied, not when the transfer is set
	// up, but a restart doesn't open it: the old transfer runs on.
	d.Hold = d.Active
	if d.Pending {
		d.Pending = false
		d.Active, d.Src, d.Pos = true, d.Next, 0
	}
}

// blocked reports whether the CPU is shut out of address a, by a running
// DMA or by the PPU, for a read or a write.
func (m *Machine) blocked(a uint16, write bool) bool {
	if a >= 0xFE00 && a < 0xFF00 && m.DMAState.Hold {
		return true
	}
	if write {
		return m.writeBlocked(a)
	}
	// The PPU has OAM to itself while it scans it (mode 2) and while it
	// draws (mode 3), and VRAM while it draws. The CPU reads 0xFF and its
	// writes are lost. The memory closes as soon as the PPU enters mode 2
	// or 3, a cycle before STAT shows it, and opens again only when STAT
	// shows mode 0. The first line after the LCD is turned on runs two
	// T-cycles late, so there it closes when STAT shows mode 3.
	p := &m.PPU
	md := p.statRead() & 3
	if in := p.mode(); !p.Fresh && (in == modeOAM || in == modeDraw) {
		md = in
	}
	return closed(a, md)
}

// writeBlocked is the PPU's side of blocked for writes, which go by STAT's
// mode bits alone, apart from the M-cycle mode 3 starts in: then even OAM
// takes writes, on all but the first line after the LCD is turned on.
func (m *Machine) writeBlocked(a uint16) bool {
	p := &m.PPU
	if !p.Fresh && p.mode() == modeDraw && p.now() == p.ModeAt {
		return false
	}
	return closed(a, p.statRead()&3)
}

// closed reports whether the PPU holds address a in mode md: OAM in modes
// 2 and 3, VRAM in mode 3.
func closed(a uint16, md uint8) bool {
	oam := a >= 0xFE00 && a < 0xFEA0
	switch md {
	case modeOAM:
		return oam
	case modeDraw:
		return oam || a >= 0x8000 && a < 0xA000
	}
	return false
}
