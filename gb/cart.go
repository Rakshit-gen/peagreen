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
	default:
		return nil, fmt.Errorf("cartridge type 0x%02X is not supported", h.Type)
	}
	c.RAM = make([]byte, h.RAMSize)
	return c, nil
}

func (c *Cart) readROM(a uint16) uint8 {
	if a < 0x4000 {
		return c.rom[a]
	}
	return c.rom[(c.ROMBank%c.banks)*0x4000+int(a-0x4000)]
}

func (c *Cart) writeROM(a uint16, v uint8) {}

func (c *Cart) readRAM(a uint16) uint8 {
	i := c.RAMBank*0x2000 + int(a-0xA000)
	if len(c.RAM) == 0 {
		return 0xFF
	}
	return c.RAM[i%len(c.RAM)]
}

func (c *Cart) writeRAM(a uint16, v uint8) {
	if len(c.RAM) == 0 {
		return
	}
	c.RAM[(c.RAMBank*0x2000+int(a-0xA000))%len(c.RAM)] = v
}
