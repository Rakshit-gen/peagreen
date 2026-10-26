package main

import (
	"math"
	"testing"
	"time"

	"github.com/Rakshit-gen/peagreen/gb"
	"github.com/Rakshit-gen/peagreen/screen"
)

func TestLayout(t *testing.T) {
	for _, c := range []struct {
		size       string
		cols, rows int
		half       bool
		col, row   int
		fails      bool
	}{
		{"auto", 160, 73, false, 1, 1, false},
		{"auto", 200, 80, false, 21, 4, false},
		{"auto", 159, 73, true, 40, 19, false},
		{"auto", 80, 37, true, 1, 1, false},
		{"auto", 79, 37, true, 0, 0, true},
		{"full", 100, 40, false, 0, 0, true},
		{"half", 200, 80, true, 61, 22, false},
		{"big", 200, 80, false, 0, 0, true},
	} {
		var p player
		err := p.layout(c.size, c.cols, c.rows)
		if (err != nil) != c.fails {
			t.Errorf("%s in %dx%d: error %v", c.size, c.cols, c.rows, err)
			continue
		}
		if !c.fails && (p.r.Half != c.half || p.r.Col != c.col || p.r.Row != c.row) {
			t.Errorf("%s in %dx%d: half %v at %d,%d, want half %v at %d,%d",
				c.size, c.cols, c.rows, p.r.Half, p.r.Col, p.r.Row, c.half, c.col, c.row)
		}
	}
}

func TestMeasure(t *testing.T) {
	p := &player{since: time.Now().Add(-2 * time.Second), ran: 240}
	p.measure()
	// 240 frames of about 16.74 ms in 2 s is about twice full speed.
	if math.Abs(p.rate-2.01) > 0.02 || p.ran != 0 {
		t.Fatalf("rate %.3f, ran %d; want about 2.01 and 0", p.rate, p.ran)
	}
}

func TestStatePath(t *testing.T) {
	p := &player{romPath: "roms/tetris.gb"}
	for slot, want := range map[int]string{0: "roms/tetris.state", 1: "roms/tetris.state", 4: "roms/tetris-4.state"} {
		p.slot = slot
		if got := p.statePath(); got != want {
			t.Errorf("slot %d: %s, want %s", slot, got, want)
		}
	}
}

func TestBlend(t *testing.T) {
	m, err := gb.New(make([]byte, 0x8000))
	if err != nil {
		t.Fatal(err)
	}
	p := &player{m: m, ghost: true}
	m.PPU.Screen[0][0] = 3
	p.blend()
	m.PPU.Screen[0][0] = 0 // a sprite flickered off
	p.blend()
	if p.picture()[0][0] != 3 {
		t.Fatal("a pixel dark in the frame before should stay dark")
	}
	p.blend()
	if p.picture()[0][0] != 0 {
		t.Fatal("two frames on, it should have faded")
	}
}

func TestFocusPause(t *testing.T) {
	p := &player{stay: true}
	p.focus(false)
	if !p.paused {
		t.Fatal("losing focus should pause")
	}
	p.focus(true)
	if p.paused {
		t.Fatal("getting focus back should carry on")
	}
	p.paused = true // paused by hand
	p.focus(false)
	p.focus(true)
	if !p.paused {
		t.Fatal("a game paused by hand should stay paused")
	}
}

func TestHelpFitsHalfSize(t *testing.T) {
	r := screen.Renderer{Half: true}
	cols, rows := r.Size()
	if len(helpText)+2 > rows {
		t.Errorf("%d lines of help, the half size frame has %d rows", len(helpText), rows)
	}
	for _, l := range helpText {
		if len(l)+4 > cols {
			t.Errorf("%q is too wide for %d columns", l, cols)
		}
	}
}
