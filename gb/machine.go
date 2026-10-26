// Package gb emulates the original Game Boy (DMG): the CPU, memory map,
// timer, video, sound, joypad and cartridge mappers.
package gb

// Interrupt bits in IE and IF.
const (
	IntVBlank = 1 << iota
	IntSTAT
	IntTimer
	IntSerial
	IntJoypad
)

// Machine is a whole Game Boy. Everything but the ROM and the cartridge
// RAM lives in the struct itself, so a copy is a snapshot.
type Machine struct {
	CPU  CPU
	Cart *Cart

	WRAM [0x2000]byte
	HRAM [0x7F]byte
	IE   uint8
	IF   uint8

	// Cycles counts T-cycles (4.194304 MHz) since power on.
	Cycles uint64
}

// New builds a machine with the cartridge in it, in the state the boot ROM
// leaves behind.
func New(rom []byte) (*Machine, error) {
	c, err := NewCart(rom)
	if err != nil {
		return nil, err
	}
	m := &Machine{Cart: c}
	m.reset()
	return m, nil
}

// reset sets what the DMG boot ROM leaves in the registers. The boot ROM
// itself is Nintendo's and isn't included.
func (m *Machine) reset() {
	c := &m.CPU
	c.A, c.F = 0x01, 0xB0
	c.B, c.C = 0x00, 0x13
	c.D, c.E = 0x00, 0xD8
	c.H, c.L = 0x01, 0x4D
	c.SP, c.PC = 0xFFFE, 0x0100
	m.IF = IntVBlank
}

// tick advances everything but the CPU by one M-cycle (4 T-cycles).
func (m *Machine) tick() {
	m.Cycles += 4
}

// Read returns the byte at a, without spending any time.
func (m *Machine) Read(a uint16) uint8 {
	switch {
	case a < 0x8000:
		return m.Cart.readROM(a)
	case a < 0xA000:
		return 0xFF
	case a < 0xC000:
		return m.Cart.readRAM(a)
	case a < 0xFE00:
		return m.WRAM[a&0x1FFF] // 0xE000-0xFDFF echoes WRAM
	case a < 0xFF00:
		return 0xFF
	case a < 0xFF80:
		return m.readIO(a)
	case a < 0xFFFF:
		return m.HRAM[a-0xFF80]
	}
	return m.IE
}

// Write stores v at a, without spending any time.
func (m *Machine) Write(a uint16, v uint8) {
	switch {
	case a < 0x8000:
		m.Cart.writeROM(a, v)
	case a < 0xA000:
	case a < 0xC000:
		m.Cart.writeRAM(a, v)
	case a < 0xFE00:
		m.WRAM[a&0x1FFF] = v
	case a < 0xFF00:
	case a < 0xFF80:
		m.writeIO(a, v)
	case a < 0xFFFF:
		m.HRAM[a-0xFF80] = v
	default:
		m.IE = v
	}
}

func (m *Machine) readIO(a uint16) uint8 {
	switch a {
	case 0xFF0F:
		return m.IF | 0xE0
	}
	return 0xFF
}

func (m *Machine) writeIO(a uint16, v uint8) {
	switch a {
	case 0xFF0F:
		m.IF = v & 0x1F
	}
}
