package main

import "testing"

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
