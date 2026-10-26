package gb

import "testing"

// blankROM is a 32 KiB ROM with a valid header and the given type.
func blankROM(typ, romCode, ramCode uint8) []byte {
	rom := make([]byte, 0x8000<<romCode)
	copy(rom[0x134:], "PEAGREEN TEST")
	rom[0x147], rom[0x148], rom[0x149] = typ, romCode, ramCode
	fixChecksums(rom)
	return rom
}

func fixChecksums(rom []byte) {
	var x uint8
	for _, b := range rom[0x134:0x14D] {
		x = x - b - 1
	}
	rom[0x14D] = x
	var sum uint16
	for i, b := range rom {
		if i != 0x14E && i != 0x14F {
			sum += uint16(b)
		}
	}
	rom[0x14E], rom[0x14F] = uint8(sum>>8), uint8(sum)
}

func TestParseHeader(t *testing.T) {
	rom := blankROM(0x13, 2, 3)
	h, err := ParseHeader(rom)
	if err != nil {
		t.Fatal(err)
	}
	if h.Title != "PEAGREEN TEST" || h.Type != 0x13 || h.ROMBanks != 8 || h.RAMSize != 32<<10 {
		t.Fatalf("got %+v", h)
	}
	if !HeaderOK(rom) || !GlobalOK(rom) {
		t.Fatal("checksums should match")
	}
	rom[0x134] ^= 1
	if HeaderOK(rom) {
		t.Fatal("header checksum should fail after changing the title")
	}
}

func TestParseHeaderRejects(t *testing.T) {
	if _, err := ParseHeader(make([]byte, 100)); err == nil {
		t.Fatal("a 100 byte rom should fail")
	}
	rom := blankROM(0, 0, 0)
	rom[0x149] = 9
	if _, err := ParseHeader(rom); err == nil {
		t.Fatal("ram code 9 should fail")
	}
}
