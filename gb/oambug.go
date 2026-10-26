package gb

// The DMG's OAM bug: while the PPU scans OAM in mode 2, a CPU access to
// 0xFE00-0xFEFF, or a 16-bit increment or decrement of a register that
// points there (the increment unit drives the address bus), scrambles the
// row of OAM the PPU is reading. OAM is 20 rows of 8 bytes, read one row
// per M-cycle, and the damage only depends on the row and on whether the
// access was a read, a write, or a read with an increment. The patterns are
// from the Pan Docs' "OAM Corruption Bug".

type oamAccess int

const (
	oamRead oamAccess = iota
	oamWrite
	oamReadInc // a read and an increment in the same M-cycle
)

// oamBug corrupts OAM if address a is in the bug's range and the PPU is
// scanning.
func (m *Machine) oamBug(a uint16, kind oamAccess) {
	p := &m.PPU
	if a < 0xFE00 || a > 0xFEFF || p.LCDC&0x80 == 0 || p.LY >= Height || p.mode() != modeOAM {
		return
	}
	row := p.Dot / 4
	if row < 1 || row >= 20 {
		return
	}
	switch kind {
	case oamWrite:
		p.corruptRow(row, func(a, b, c uint16) uint16 { return (a^c)&(b^c) ^ c })
	case oamRead:
		p.corruptRow(row, func(a, b, c uint16) uint16 { return b | a&c })
	case oamReadInc:
		if row >= 4 && row < 19 {
			a, b, c, d := p.oamWord(row-2, 0), p.oamWord(row-1, 0), p.oamWord(row, 0), p.oamWord(row-1, 2)
			p.setOAMWord(row-1, 0, b&(a|c|d)|a&c&d)
			prev := p.OAM[(row-1)*8 : row*8]
			copy(p.OAM[row*8:], prev)
			copy(p.OAM[(row-2)*8:], prev)
		}
		p.corruptRow(row, func(a, b, c uint16) uint16 { return b | a&c })
	}
}

// corruptRow replaces the row's first word with f(a, b, c), a being that
// word, b the previous row's first and c its third, and copies the rest of
// the previous row over the rest of this one.
func (p *PPU) corruptRow(row int, f func(a, b, c uint16) uint16) {
	a, b, c := p.oamWord(row, 0), p.oamWord(row-1, 0), p.oamWord(row-1, 2)
	p.setOAMWord(row, 0, f(a, b, c))
	copy(p.OAM[row*8+2:row*8+8], p.OAM[(row-1)*8+2:row*8])
}

func (p *PPU) oamWord(row, i int) uint16 {
	j := row*8 + i*2
	return uint16(p.OAM[j]) | uint16(p.OAM[j+1])<<8
}

func (p *PPU) setOAMWord(row, i int, v uint16) {
	j := row*8 + i*2
	p.OAM[j], p.OAM[j+1] = uint8(v), uint8(v>>8)
}
