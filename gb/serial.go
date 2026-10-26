package gb

import "io"

// Serial is the link port with no cable in it. Bytes the game sends go to
// a writer, which is how blargg's test ROMs report, and what comes back is
// 0xFF, as from an empty port.
type Serial struct {
	SB, SC uint8
	Left   int  // bits left to shift in the transfer in progress
	Clock  bool // the serial clock's last level
}

// serialBit is the bit of the system counter the internal serial clock
// is taken from: 8192 Hz. Because it's the same counter DIV shows, clock
// edges line up with the time since power on, not with the write to SC.
const serialBit = 0x100

// SetSerial sends every byte the game shifts out of the link port to w.
func (m *Machine) SetSerial(w io.Writer) { m.serialOut = w }

func (m *Machine) serialTick() {
	s := &m.Serial
	// The transfer ends, and the interrupt fires, an M-cycle before the
	// counter's edge shows in DIV.
	clock := (m.Timer.Div+4)&serialBit != 0
	fell := s.Clock && !clock
	s.Clock = clock
	if s.Left <= 0 || !fell {
		return
	}
	if s.Left--; s.Left > 0 {
		return
	}
	if m.serialOut != nil {
		m.serialOut.Write([]byte{s.SB})
	}
	s.SB = 0xFF
	s.SC &^= 0x80
	m.IF |= IntSerial
}

func (m *Machine) writeSC(v uint8) {
	s := &m.Serial
	s.SC = v
	// Only the internal clock moves without a partner: 8 bits at 8192 Hz.
	if v&0x81 == 0x81 {
		s.Left = 8
	}
}
