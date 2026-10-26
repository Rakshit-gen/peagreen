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
		m.renderWindow(&bg)
	}
	for x := range Width {
		p.Screen[ly][x] = shade(p.BGP, bg[x])
	}
	if p.LCDC&2 != 0 {
		m.renderSprites(&bg)
	}
}

// renderSprites draws up to ten sprites on line LY. On the DMG the sprite
// with the lower X wins, then the one earlier in OAM. A winning pixel
// hides the sprites below it even when the background then hides it.
func (m *Machine) renderSprites(bg *[Width]uint8) {
	p := &m.PPU
	ly := int(p.LY)
	h := 8
	if p.LCDC&4 != 0 {
		h = 16
	}
	var line [10]int // OAM indexes, in OAM order
	n := 0
	for i := 0; i < 40 && n < 10; i++ {
		y := int(p.OAM[i*4]) - 16
		if ly >= y && ly < y+h {
			line[n] = i
			n++
		}
	}
	// Insertion sort by X keeps OAM order for ties.
	sel := line[:n]
	for i := 1; i < n; i++ {
		for j := i; j > 0 && p.OAM[sel[j]*4+1] < p.OAM[sel[j-1]*4+1]; j-- {
			sel[j], sel[j-1] = sel[j-1], sel[j]
		}
	}
	var taken [Width]bool
	for _, i := range sel {
		o := p.OAM[i*4 : i*4+4]
		y, x, tile, attr := int(o[0])-16, int(o[1])-8, o[2], o[3]
		row := ly - y
		if attr&0x40 != 0 {
			row = h - 1 - row
		}
		if h == 16 {
			tile &^= 1
		}
		pal := p.OBP0
		if attr&0x10 != 0 {
			pal = p.OBP1
		}
		for px := range 8 {
			sx := x + px
			if sx < 0 || sx >= Width || taken[sx] {
				continue
			}
			col := px
			if attr&0x20 != 0 {
				col = 7 - px
			}
			c := p.tilePixel(tile+uint8(row/8), col, row&7, true)
			if c == 0 {
				continue
			}
			taken[sx] = true
			if attr&0x80 != 0 && bg[sx] != 0 {
				continue
			}
			p.Screen[ly][sx] = shade(pal, c)
		}
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

// renderWindow draws the window over the background. It has its own line
// counter, which only moves on lines where the window actually showed.
func (m *Machine) renderWindow(bg *[Width]uint8) {
	p := &m.PPU
	if p.LCDC&0x20 == 0 || !p.WinY || p.WX > 166 {
		return
	}
	tileMap := 0x1800
	if p.LCDC&0x40 != 0 {
		tileMap = 0x1C00
	}
	unsigned := p.LCDC&0x10 != 0
	y := p.WinLine
	row := tileMap + y/8*32
	for x := max(0, int(p.WX)-7); x < Width; x++ {
		wx := x - (int(p.WX) - 7)
		bg[x] = p.tilePixel(p.VRAM[row+wx/8], wx&7, y&7, unsigned)
	}
	p.WinLine++
}
