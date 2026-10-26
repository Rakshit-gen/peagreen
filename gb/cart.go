package gb

import "fmt"

// Mapper kinds.
const (
	mapNone = iota
	mapMBC1
	mapMBC2
	mapMBC3
	mapMBC5
)

// Cart is a cartridge: its ROM, its RAM and the mapper's registers. All
// mappers share one struct so a machine can be copied by value for save
// states and rewind.
type Cart struct {
	rom    []byte
	banks  int
	Header Header
	kind   int

	Battery bool
	RAM     []byte
	RAMOn   bool
	ROMBank int // bank mapped at 0x4000
	RAMBank int

	// MBC1 splits the bank number over two registers, and a mode bit
	// decides whether the upper one also banks 0x0000 and RAM.
	Bank1, Bank2 uint8
	Mode         uint8

	rumble bool
	multi  bool // MBC1 multicart, with bank bit 4 not wired

	HasRTC bool
	RTC    RTC
}

// NewCart wraps a ROM image.
func NewCart(rom []byte) (*Cart, error) {
	h, err := ParseHeader(rom)
	if err != nil {
		return nil, err
	}
	if len(rom)%0x4000 != 0 {
		return nil, errShort
	}
	c := &Cart{rom: rom, banks: len(rom) / 0x4000, Header: h, ROMBank: 1}
	switch h.Type {
	case 0x00:
		c.kind = mapNone
	case 0x08, 0x09:
		c.kind = mapNone
		c.Battery = h.Type == 0x09
	case 0x01, 0x02, 0x03:
		c.kind = mapMBC1
		c.Battery = h.Type == 0x03
		c.Bank1 = 1
		c.multi = isMulticart(rom)
	case 0x05, 0x06:
		c.kind = mapMBC2
		c.Battery = h.Type == 0x06
	case 0x0F, 0x10, 0x11, 0x12, 0x13:
		c.kind = mapMBC3
		c.Battery = h.Type == 0x0F || h.Type == 0x10 || h.Type == 0x13
		c.HasRTC = h.Type == 0x0F || h.Type == 0x10
	case 0x19, 0x1A, 0x1B, 0x1C, 0x1D, 0x1E:
		// 0x1C-0x1E have a rumble motor on RAM bank bit 3, which is
		// ignored here apart from not counting it as a bank bit.
		c.kind = mapMBC5
		c.Battery = h.Type == 0x1B || h.Type == 0x1E
		c.rumble = h.Type >= 0x1C
	default:
		return nil, fmt.Errorf("cartridge type 0x%02X is not supported", h.Type)
	}
	c.RAM = make([]byte, h.RAMSize)
	if c.kind == mapMBC2 {
		c.RAM = make([]byte, 512)
	}
	return c, nil
}

func (c *Cart) readROM(a uint16) uint8 {
	if a < 0x4000 {
		bank := 0
		if c.kind == mapMBC1 && c.Mode == 1 {
			bank = int(c.Bank2) << c.bank2Shift()
		}
		return c.rom[(bank%c.banks)*0x4000+int(a)]
	}
	return c.rom[(c.ROMBank%c.banks)*0x4000+int(a-0x4000)]
}

// writeROM is a write to 0x0000-0x7FFF, which goes to the mapper.
func (c *Cart) writeROM(a uint16, v uint8) {
	switch c.kind {
	case mapMBC1:
		switch {
		case a < 0x2000:
			c.RAMOn = v&0x0F == 0x0A
		case a < 0x4000:
			c.Bank1 = v & 0x1F
			if c.Bank1 == 0 {
				c.Bank1 = 1
			}
		case a < 0x6000:
			c.Bank2 = v & 3
		default:
			c.Mode = v & 1
		}
		if c.multi {
			c.ROMBank = int(c.Bank2)<<4 | int(c.Bank1&0x0F)
		} else {
			c.ROMBank = int(c.Bank2)<<5 | int(c.Bank1)
		}
		c.RAMBank = 0
		if c.Mode == 1 {
			c.RAMBank = int(c.Bank2)
		}
	case mapMBC3:
		switch {
		case a < 0x2000:
			c.RAMOn = v&0x0F == 0x0A
		case a < 0x4000:
			c.ROMBank = int(v & 0x7F)
			if c.ROMBank == 0 {
				c.ROMBank = 1
			}
		case a < 0x6000:
			c.RAMBank = int(v & 0x0F)
		case c.HasRTC:
			if v == 0 {
				c.RTC.Armed = true
			} else if v == 1 && c.RTC.Armed {
				c.RTC.latch()
				c.RTC.Armed = false
			} else {
				c.RTC.Armed = false
			}
		}
	case mapMBC5:
		switch {
		case a < 0x2000:
			c.RAMOn = v&0x0F == 0x0A
		case a < 0x3000:
			// Unlike the others, MBC5 can map bank 0 at 0x4000.
			c.ROMBank = c.ROMBank&0x100 | int(v)
		case a < 0x4000:
			c.ROMBank = c.ROMBank&0xFF | int(v&1)<<8
		case a < 0x6000:
			c.RAMBank = int(v & 0x0F)
			if c.rumble {
				c.RAMBank &= 7
			}
		}
	case mapMBC2:
		// Address bit 8 picks the register for the whole 0x0000-0x3FFF range.
		if a >= 0x4000 {
			return
		}
		if a&0x100 == 0 {
			c.RAMOn = v&0x0F == 0x0A
		} else {
			c.ROMBank = int(v & 0x0F)
			if c.ROMBank == 0 {
				c.ROMBank = 1
			}
		}
	}
}

func (c *Cart) bank2Shift() int {
	if c.multi {
		return 4
	}
	return 5
}

// isMulticart guesses whether an MBC1 ROM is a multicart, which wires
// the upper bank bits one place lower. Nothing in the header says so, but
// every game in one has its own Nintendo logo, and so does bank 0x10.
func isMulticart(rom []byte) bool {
	return len(rom) == 1<<20 && string(rom[0x104:0x134]) == string(rom[0x40104:0x40134])
}

// ramOn reports whether cartridge RAM answers. Mapperless carts have no
// enable register.
func (c *Cart) ramOn() bool { return c.kind == mapNone || c.RAMOn }

// rtcSelected reports whether 0xA000-0xBFFF shows a clock register.
func (c *Cart) rtcSelected() bool { return c.HasRTC && c.RAMBank >= 8 && c.RAMBank <= 12 }

func (c *Cart) readRAM(a uint16) uint8 {
	if c.kind == mapMBC2 {
		if !c.RAMOn {
			return 0xFF
		}
		// 512 half-bytes, repeated over the whole range. The top half of
		// each byte isn't wired and reads as ones.
		return c.RAM[a&0x1FF] | 0xF0
	}
	if c.rtcSelected() {
		if !c.RAMOn {
			return 0xFF
		}
		return c.RTC.read(c.RAMBank)
	}
	i := c.RAMBank*0x2000 + int(a-0xA000)
	if len(c.RAM) == 0 || !c.ramOn() {
		return 0xFF
	}
	return c.RAM[i%len(c.RAM)]
}

func (c *Cart) writeRAM(a uint16, v uint8) {
	if c.kind == mapMBC2 {
		if c.RAMOn {
			c.RAM[a&0x1FF] = v & 0x0F
		}
		return
	}
	if c.rtcSelected() {
		if c.RAMOn {
			c.RTC.write(c.RAMBank, v)
		}
		return
	}
	if len(c.RAM) == 0 || !c.ramOn() {
		return
	}
	c.RAM[(c.RAMBank*0x2000+int(a-0xA000))%len(c.RAM)] = v
}
