package gb

// Flag bits in F.
const (
	flagZ = 0x80
	flagN = 0x40
	flagH = 0x20
	flagC = 0x10
)

// CPU is the SM83 core's registers and state.
type CPU struct {
	A, F, B, C, D, E, H, L uint8
	SP, PC                 uint16

	IME       bool // interrupts enabled
	EIPending bool // EI takes effect after the next instruction
	Halted    bool
	HaltBug   bool // the next fetch doesn't move PC
	Locked    bool // an undefined opcode hangs the CPU for good
}

func (c *CPU) bc() uint16 { return uint16(c.B)<<8 | uint16(c.C) }
func (c *CPU) de() uint16 { return uint16(c.D)<<8 | uint16(c.E) }
func (c *CPU) hl() uint16 { return uint16(c.H)<<8 | uint16(c.L) }
func (c *CPU) af() uint16 { return uint16(c.A)<<8 | uint16(c.F) }

func (c *CPU) setBC(v uint16) { c.B, c.C = uint8(v>>8), uint8(v) }
func (c *CPU) setDE(v uint16) { c.D, c.E = uint8(v>>8), uint8(v) }
func (c *CPU) setHL(v uint16) { c.H, c.L = uint8(v>>8), uint8(v) }
func (c *CPU) setAF(v uint16) { c.A, c.F = uint8(v>>8), uint8(v)&0xF0 }

func (c *CPU) flag(f uint8) bool { return c.F&f != 0 }

// setFlags sets all four flags at once.
func (c *CPU) setFlags(z, n, h, cy bool) {
	c.F = 0
	if z {
		c.F |= flagZ
	}
	if n {
		c.F |= flagN
	}
	if h {
		c.F |= flagH
	}
	if cy {
		c.F |= flagC
	}
}
