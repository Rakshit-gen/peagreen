package gb

import (
	"errors"
	"fmt"
	"strings"
)

// Header is what a cartridge says about itself at 0x0100-0x014F.
type Header struct {
	Title    string
	Type     uint8 // 0x0147: mapper and extras
	ROMBanks int   // 16 KiB each
	RAMSize  int   // bytes of cartridge RAM, not counting MBC2's built-in RAM
	CGB      uint8 // 0x0143: 0x80 works on both, 0xC0 is Color only
	Checksum uint8 // 0x014D
	Global   uint16
}

// ParseHeader reads the header of a ROM image.
func ParseHeader(rom []byte) (Header, error) {
	if len(rom) < 0x8000 {
		return Header{}, fmt.Errorf("rom is %d bytes, shorter than the smallest cartridge (32 KiB)", len(rom))
	}
	var h Header
	h.CGB = rom[0x143]
	title := rom[0x134:0x144]
	if h.CGB&0x80 != 0 {
		// Color-era carts reuse the end of the title for a maker code and the
		// CGB flag.
		title = rom[0x134:0x13F]
	}
	h.Title = strings.TrimRight(strings.Map(func(r rune) rune {
		if r < 0x20 || r > 0x7E {
			return -1
		}
		return r
	}, string(title)), " ")
	h.Type = rom[0x147]
	if rom[0x148] > 8 {
		return h, fmt.Errorf("rom size code 0x%02X is not one I know", rom[0x148])
	}
	h.ROMBanks = 2 << rom[0x148]
	switch rom[0x149] {
	case 0:
	case 1:
		h.RAMSize = 2 << 10
	case 2:
		h.RAMSize = 8 << 10
	case 3:
		h.RAMSize = 32 << 10
	case 4:
		h.RAMSize = 128 << 10
	case 5:
		h.RAMSize = 64 << 10
	default:
		return h, fmt.Errorf("ram size code 0x%02X is not one I know", rom[0x149])
	}
	h.Checksum = rom[0x14D]
	h.Global = uint16(rom[0x14E])<<8 | uint16(rom[0x14F])
	return h, nil
}

// HeaderOK reports whether the header checksum matches. The boot ROM
// refuses to start a cartridge when it doesn't.
func HeaderOK(rom []byte) bool {
	var x uint8
	for _, b := range rom[0x134:0x14D] {
		x = x - b - 1
	}
	return x == rom[0x14D]
}

// GlobalOK reports whether the checksum over the whole ROM matches. Real
// hardware never checks it, and plenty of homebrew gets it wrong.
func GlobalOK(rom []byte) bool {
	var sum uint16
	for i, b := range rom {
		if i != 0x14E && i != 0x14F {
			sum += uint16(b)
		}
	}
	return sum == uint16(rom[0x14E])<<8|uint16(rom[0x14F])
}

var errShort = errors.New("rom is not a whole number of 16 KiB banks")
