package gb

// tilePixel is the 2-bit colour of pixel (x, y) in a tile, where the tile
// number is read the way LCDC bit 4 says: unsigned from 0x8000, or signed
// from 0x9000.
func (p *PPU) tilePixel(tile uint8, x, y int, unsigned bool) uint8 {
	base := 0x1000 + int(int8(tile))*16
	if unsigned {
		base = int(tile) * 16
	}
	lo, hi := p.VRAM[base+y*2], p.VRAM[base+y*2+1]
	bit := 7 - x
	return (hi>>bit&1)<<1 | lo>>bit&1
}

func shade(palette, color uint8) uint8 { return palette >> (color * 2) & 3 }

// renderLine draws line LY into Screen.
func (m *Machine) renderLine() {
	p := &m.PPU
	ly := int(p.LY)
	// bg holds the raw background colour of each pixel, which sprites with
	// the priority bit check against.
	var bg [Width]uint8
	if p.LCDC&1 != 0 {
		m.renderBG(&bg)
	}
	for x := range Width {
		p.Screen[ly][x] = shade(p.BGP, bg[x])
	}
}

func (m *Machine) renderBG(bg *[Width]uint8) {
	p := &m.PPU
	tileMap := 0x1800
	if p.LCDC&0x08 != 0 {
		tileMap = 0x1C00
	}
	unsigned := p.LCDC&0x10 != 0
	y := (int(p.LY) + int(p.SCY)) & 255
	row := tileMap + y/8*32
	for x := range Width {
		bx := (x + int(p.SCX)) & 255
		bg[x] = p.tilePixel(p.VRAM[row+bx/8], bx&7, y&7, unsigned)
	}
}
