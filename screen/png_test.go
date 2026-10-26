package screen

import (
	"bytes"
	"image/png"
	"testing"

	"github.com/Rakshit-gen/peagreen/gb"
)

func TestPNG(t *testing.T) {
	var frame [gb.Height][gb.Width]uint8
	frame[0][1] = 3
	var buf bytes.Buffer
	if err := WritePNG(&buf, &frame, Palettes[0], 2); err != nil {
		t.Fatal(err)
	}
	img, err := png.Decode(&buf)
	if err != nil {
		t.Fatal(err)
	}
	if b := img.Bounds(); b.Dx() != 320 || b.Dy() != 288 {
		t.Fatalf("size %v, want 320x288", b)
	}
	dark := Palettes[0].Colors[3]
	for _, xy := range [][2]int{{2, 0}, {3, 1}} {
		r, g, b, _ := img.At(xy[0], xy[1]).RGBA()
		if uint8(r>>8) != dark.R || uint8(g>>8) != dark.G || uint8(b>>8) != dark.B {
			t.Fatalf("pixel %v isn't the darkest shade", xy)
		}
	}
}

func TestFind(t *testing.T) {
	if p, err := Find("POCKET"); err != nil || p.Name != "pocket" {
		t.Fatalf("got %v, %v", p.Name, err)
	}
	if _, err := Find("purple"); err == nil {
		t.Fatal("purple shouldn't exist")
	}
}
