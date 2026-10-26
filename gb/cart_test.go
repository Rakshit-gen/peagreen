package gb

import "testing"

// bankedCart is a cartridge whose every bank starts with its own number.
func bankedCart(t *testing.T, typ, romCode, ramCode uint8) *Cart {
	t.Helper()
	rom := blankROM(typ, romCode, ramCode)
	for b := 1; b < len(rom)/0x4000; b++ {
		rom[b*0x4000] = uint8(b)
	}
	fixChecksums(rom)
	c, err := NewCart(rom)
	if err != nil {
		t.Fatal(err)
	}
	return c
}

func TestMBC1Banks(t *testing.T) {
	c := bankedCart(t, 0x03, 6, 3) // 2 MiB, 32 KiB RAM
	if got := c.readROM(0x4000); got != 1 {
		t.Fatalf("bank at start = %d, want 1", got)
	}
	c.writeROM(0x2000, 0) // 0 maps to 1
	if got := c.readROM(0x4000); got != 1 {
		t.Fatalf("bank 0 written: got %d, want 1", got)
	}
	c.writeROM(0x2000, 5)
	c.writeROM(0x4000, 2)
	if got := c.readROM(0x4000); got != 0x45 {
		t.Fatalf("bank = %d, want 0x45", got)
	}
	if got := c.readROM(0x0000); got != 0 {
		t.Fatalf("mode 0: 0x0000 shows bank %d, want 0", got)
	}
	c.writeROM(0x6000, 1)
	if got := c.readROM(0x0000); got != 0x40 {
		t.Fatalf("mode 1: 0x0000 shows bank %d, want 0x40", got)
	}
}

func TestMBC1RAM(t *testing.T) {
	c := bankedCart(t, 0x03, 2, 3)
	c.writeRAM(0xA000, 7)
	if got := c.readRAM(0xA000); got != 0xFF {
		t.Fatalf("disabled RAM read %d, want 0xFF", got)
	}
	c.writeROM(0x0000, 0x0A)
	c.writeROM(0x6000, 1)
	c.writeROM(0x4000, 2)
	c.writeRAM(0xA000, 7)
	c.writeROM(0x4000, 0)
	if got := c.readRAM(0xA000); got != 0 {
		t.Fatalf("bank 0 = %d, want 0", got)
	}
	c.writeROM(0x4000, 2)
	if got := c.readRAM(0xA000); got != 7 {
		t.Fatalf("bank 2 = %d, want 7", got)
	}
}

func TestMBC2(t *testing.T) {
	c := bankedCart(t, 0x06, 3, 0)
	c.writeROM(0x2100, 3) // bit 8 set: ROM bank
	if got := c.readROM(0x4000); got != 3 {
		t.Fatalf("bank = %d, want 3", got)
	}
	c.writeROM(0x0000, 0x0A) // bit 8 clear: RAM enable
	c.writeRAM(0xA005, 0xAB)
	if got := c.readRAM(0xA205); got != 0xFB {
		t.Fatalf("echoed nibble = %#x, want 0xFB", got)
	}
}

func TestMBC3Clock(t *testing.T) {
	c := bankedCart(t, 0x10, 2, 3)
	c.writeROM(0x0000, 0x0A)
	c.RTC.H, c.RTC.M, c.RTC.S, c.RTC.Days = 23, 59, 59, 511
	c.RTC.tick(cyclesPerSecond)
	c.writeROM(0x6000, 0)
	c.writeROM(0x6000, 1)
	read := func(reg int) uint8 {
		c.writeROM(0x4000, uint8(reg))
		return c.readRAM(0xA000)
	}
	if s, m, h, dl, dh := read(8), read(9), read(10), read(11), read(12); s|m|h|dl != 0 || dh != 0x80 {
		t.Fatalf("after rollover: %d:%d:%d day %d dh %#x", h, m, s, dl, dh)
	}
	// The latched value stays put while the clock runs on.
	c.RTC.tick(5 * cyclesPerSecond)
	if s := read(8); s != 0 {
		t.Fatalf("latched seconds moved to %d", s)
	}
	c.writeROM(0x6000, 0)
	c.writeROM(0x6000, 1)
	if s := read(8); s != 5 {
		t.Fatalf("seconds after relatch = %d, want 5", s)
	}
}

func TestRTCOutOfRange(t *testing.T) {
	var r RTC
	r.write(8, 61)
	r.tick(3 * cyclesPerSecond) // 62, 63, then 0 without a minute
	if r.S != 0 || r.M != 0 {
		t.Fatalf("got %d:%d, want 0:0", r.M, r.S)
	}
}

func TestMBC5Banks(t *testing.T) {
	c := bankedCart(t, 0x1B, 8, 4) // 8 MiB
	c.writeROM(0x2000, 0)
	if got := c.readROM(0x4000); got != 0 {
		t.Fatalf("MBC5 should map bank 0, got %d", got)
	}
	c.writeROM(0x2000, 0x34)
	c.writeROM(0x3000, 1)
	if got := c.ROMBank; got != 0x134 {
		t.Fatalf("bank = %#x, want 0x134", got)
	}
	if got := c.readROM(0x4000); got != 0x34 {
		t.Fatalf("first byte = %#x, want 0x34 (bank number mod 256)", got)
	}
}
