package screen

import (
	"bytes"
	"image/color"
	"strings"
	"testing"

	"github.com/Rakshit-gen/peagreen/gb"
)

func TestRendererDiffs(t *testing.T) {
	var frame [gb.Height][gb.Width]uint8
	r := &Renderer{Palette: Palettes[0], Col: 3, Row: 2}
	first := r.Frame(&frame)
	if !bytes.HasPrefix(first, []byte("\x1b[2;3H")) {
		t.Fatalf("the first frame should start by moving to its corner: %q", first[:20])
	}
	if n := strings.Count(string(first), "▀"); n != 160*72 {
		t.Fatalf("first frame drew %d cells, want all %d", n, 160*72)
	}
	if again := r.Frame(&frame); len(again) != 0 {
		t.Fatalf("an unchanged frame wrote %d bytes", len(again))
	}
	frame[5][10] = 3 // row 5 is the bottom half of cell row 2
	out := string(r.Frame(&frame))
	if strings.Count(out, "▀") != 1 || !strings.Contains(out, "\x1b[4;13H") {
		t.Fatalf("one changed pixel gave %q", out)
	}
	dark := Palettes[0].Colors[3]
	if !strings.Contains(out, "48;2;15;56;15m") || dark.R != 15 {
		t.Fatalf("the bottom pixel should set the dark background: %q", out)
	}
	r.Invalidate()
	if n := bytes.Count(r.Frame(&frame), []byte("▀")); n != 160*72 {
		t.Fatalf("after Invalidate drew %d cells", n)
	}
}

func TestRendererHalf(t *testing.T) {
	var frame [gb.Height][gb.Width]uint8
	frame[1][1] = 2 // inside the top 2x2 of cell (0, 0)
	r := &Renderer{Palette: Palettes[0], Half: true, Col: 1, Row: 1}
	if c, rows := r.Size(); c != 80 || rows != 36 {
		t.Fatalf("half size %dx%d", c, rows)
	}
	if c := r.cell(&frame, 0, 0); c != (cell{2, 0}) {
		t.Fatalf("cell = %+v, want the darkest shade on top", c)
	}
}

func TestXterm256(t *testing.T) {
	for _, c := range []struct {
		in   color.RGBA
		want int
	}{
		{color.RGBA{0, 0, 0, 255}, 16},
		{color.RGBA{255, 255, 255, 255}, 231},
		{color.RGBA{255, 0, 0, 255}, 196},
		{color.RGBA{128, 128, 128, 255}, 244}, // the grey ramp is closer than the cube
		{color.RGBA{0x0F, 0x38, 0x0F, 255}, 22},
	} {
		if got := xterm256(c.in); got != c.want {
			t.Errorf("xterm256(%v) = %d, want %d", c.in, got, c.want)
		}
	}
}

func TestRenderer256(t *testing.T) {
	var frame [gb.Height][gb.Width]uint8
	r := &Renderer{Palette: Palettes[0], Colors: 256, Col: 1, Row: 1}
	out := string(r.Frame(&frame))
	if strings.Contains(out, ";2;") || !strings.Contains(out, "\x1b[38;5;") {
		t.Fatalf("256-colour output should use only 5; codes: %q", out[:40])
	}
}
