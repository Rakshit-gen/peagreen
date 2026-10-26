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
	case 0x05, 0x06:
		c.kind = mapMBC2
		c.Battery = h.Type == 0x06
	case 0x0F, 0x10, 0x11, 0x12, 0x13:
		c.kind = mapMBC3
		c.Battery = h.Type == 0x0F || h.Type == 0x10 || h.Type == 0x13
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
			bank = int(c.Bank2) << 5
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
		c.ROMBank = int(c.Bank2)<<5 | int(c.Bank1)
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

// ramOn reports whether cartridge RAM answers. Mapperless carts have no
// enable register.
func (c *Cart) ramOn() bool { return c.kind == mapNone || c.RAMOn }

func (c *Cart) readRAM(a uint16) uint8 {
	if c.kind == mapMBC2 {
		if !c.RAMOn {
			return 0xFF
		}
		// 512 half-bytes, repeated over the whole range. The top half of
		// each byte isn't wired and reads as ones.
		return c.RAM[a&0x1FF] | 0xF0
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
	if len(c.RAM) == 0 || !c.ramOn() {
		return
	}
	c.RAM[(c.RAMBank*0x2000+int(a-0xA000))%len(c.RAM)] = v
}
