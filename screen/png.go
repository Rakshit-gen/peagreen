package screen

import (
	"image"
	"image/color"
	"image/png"
	"io"

	"github.com/Rakshit-gen/peagreen/gb"
)

// Image draws a frame in a palette, each pixel scale times over.
func Image(frame *[gb.Height][gb.Width]uint8, p Palette, scale int) *image.Paletted {
	scale = max(scale, 1)
	pal := color.Palette{p.Colors[0], p.Colors[1], p.Colors[2], p.Colors[3]}
	img := image.NewPaletted(image.Rect(0, 0, gb.Width*scale, gb.Height*scale), pal)
	for y := range gb.Height * scale {
		row := img.Pix[y*img.Stride:]
		for x := range gb.Width * scale {
			row[x] = frame[y/scale][x/scale]
		}
	}
	return img
}

// WritePNG writes a frame as a PNG.
func WritePNG(w io.Writer, frame *[gb.Height][gb.Width]uint8, p Palette, scale int) error {
	return png.Encode(w, Image(frame, p, scale))
}
