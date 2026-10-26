package screen

import (
	"bytes"
	"strconv"

	"github.com/Rakshit-gen/peagreen/gb"
)

// Renderer draws frames in a terminal with the upper half block "▀": its
// foreground colour is the top pixel and its background the bottom one,
// so a 160x144 frame takes 160 columns and 72 rows. Half draws it in 80x36
// instead, for smaller terminals. Only cells that changed are redrawn.
type Renderer struct {
	Palette Palette
	Half    bool
	Col     int // where the frame's top left cell goes, from 1
	Row     int

	prev     []cell
	valid    bool
	buf      bytes.Buffer
	fg, bg   int8 // colours last set, -1 for unknown
	cx, cy   int  // where the cursor is, -1 for unknown
	palCodes [2][4]string
}

type cell struct{ top, bot uint8 }

// Size is the frame's size in cells.
func (r *Renderer) Size() (cols, rows int) {
	if r.Half {
		return gb.Width / 2, gb.Height / 4
	}
	return gb.Width, gb.Height / 2
}

// Invalidate makes the next frame redraw every cell, after the screen was
// cleared or the palette changed.
func (r *Renderer) Invalidate() { r.valid = false }

// Frame returns what to write to the terminal to show frame.
func (r *Renderer) Frame(frame *[gb.Height][gb.Width]uint8) []byte {
	cols, rows := r.Size()
	if len(r.prev) != cols*rows {
		r.prev = make([]cell, cols*rows)
		r.valid = false
	}
	if !r.valid {
		r.codes()
	}
	r.buf.Reset()
	r.fg, r.bg, r.cx, r.cy = -1, -1, -1, -1
	for y := range rows {
		for x := range cols {
			c := r.cell(frame, x, y)
			i := y*cols + x
			if r.valid && r.prev[i] == c {
				continue
			}
			r.prev[i] = c
			r.put(x, y, c)
		}
	}
	r.valid = true
	if r.buf.Len() > 0 {
		r.buf.WriteString("\x1b[0m")
	}
	return r.buf.Bytes()
}

// cell is the pair of shades for cell (x, y). In half size a cell covers
// 2x4 pixels, and each half shows the darkest of its 2x2, so thin dark
// lines like text survive.
func (r *Renderer) cell(frame *[gb.Height][gb.Width]uint8, x, y int) cell {
	if !r.Half {
		return cell{frame[y*2][x], frame[y*2+1][x]}
	}
	dark := func(px, py int) uint8 {
		return max(frame[py][px], frame[py][px+1], frame[py+1][px], frame[py+1][px+1])
	}
	return cell{dark(x*2, y*4), dark(x*2, y*4+2)}
}

func (r *Renderer) put(x, y int, c cell) {
	if r.cy != y || r.cx != x {
		r.buf.WriteString("\x1b[")
		r.buf.WriteString(strconv.Itoa(r.Row + y))
		r.buf.WriteByte(';')
		r.buf.WriteString(strconv.Itoa(r.Col + x))
		r.buf.WriteByte('H')
	}
	if int8(c.top) != r.fg {
		r.buf.WriteString(r.palCodes[0][c.top])
		r.fg = int8(c.top)
	}
	if int8(c.bot) != r.bg {
		r.buf.WriteString(r.palCodes[1][c.bot])
		r.bg = int8(c.bot)
	}
	r.buf.WriteString("▀")
	r.cx, r.cy = x+1, y
}

// codes works out the colour escapes for the palette.
func (r *Renderer) codes() {
	for i, c := range r.Palette.Colors {
		rgb := strconv.Itoa(int(c.R)) + ";" + strconv.Itoa(int(c.G)) + ";" + strconv.Itoa(int(c.B)) + "m"
		r.palCodes[0][i] = "\x1b[38;2;" + rgb
		r.palCodes[1][i] = "\x1b[48;2;" + rgb
	}
}
