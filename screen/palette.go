// Package screen turns the Game Boy's four shades into colours, for the
// terminal and for PNG files.
package screen

import (
	"fmt"
	"image/color"
	"strings"
)

// Palette maps shades 0 (lightest) to 3 to colours.
type Palette struct {
	Name   string
	Colors [4]color.RGBA
}

func rgb(v uint32) color.RGBA {
	return color.RGBA{uint8(v >> 16), uint8(v >> 8), uint8(v), 0xFF}
}

// Palettes are the built-in looks, the first being the default.
var Palettes = []Palette{
	// The pea-soup green of the original screen.
	{"green", [4]color.RGBA{rgb(0x9BBC0F), rgb(0x8BAC0F), rgb(0x306230), rgb(0x0F380F)}},
	// The Game Boy Pocket's greyer, sharper screen.
	{"pocket", [4]color.RGBA{rgb(0xC4CFA1), rgb(0x8B956D), rgb(0x4D533C), rgb(0x1F1F1F)}},
	// Plain greys, as the hardware reports them.
	{"gray", [4]color.RGBA{rgb(0xFFFFFF), rgb(0xAAAAAA), rgb(0x555555), rgb(0x000000)}},
	// An amber monochrome monitor.
	{"amber", [4]color.RGBA{rgb(0xFFB000), rgb(0xC07800), rgb(0x704000), rgb(0x201000)}},
}

// Find returns the palette with a name, ignoring case.
func Find(name string) (Palette, error) {
	var names []string
	for _, p := range Palettes {
		if strings.EqualFold(p.Name, name) {
			return p, nil
		}
		names = append(names, p.Name)
	}
	return Palette{}, fmt.Errorf("no palette %q; there's %s", name, strings.Join(names, ", "))
}
