package gb

// Timer is DIV and TIMA. DIV is the top byte of a 16-bit counter that
// runs at the full clock; TIMA counts falling edges of one bit of that
// counter, picked by TAC.
type Timer struct {
	Div  uint16
	TIMA uint8
	TMA  uint8
	TAC  uint8
}

// timerBits is the counter bit TIMA watches, for each TAC clock select.
var timerBits = [4]uint16{1 << 9, 1 << 3, 1 << 5, 1 << 7}

// signal is the input TIMA counts falling edges of.
func (t *Timer) signal() bool {
	return t.TAC&4 != 0 && t.Div&timerBits[t.TAC&3] != 0
}

func (m *Machine) timerTick() {
	t := &m.Timer
	for range 4 {
		before := t.signal()
		t.Div++
		if before && !t.signal() {
			m.timaInc()
		}
	}
}

func (m *Machine) timaInc() {
	t := &m.Timer
	if t.TIMA++; t.TIMA == 0 {
		t.TIMA = t.TMA
		m.IF |= IntTimer
	}
}

// timerWrite handles writes to DIV and TAC, which can drop the signal
// and count one more than expected.
func (m *Machine) timerWrite(a uint16, v uint8) {
	t := &m.Timer
	before := t.signal()
	switch a {
	case 0xFF04:
		t.Div = 0
	case 0xFF05:
		t.TIMA = v
	case 0xFF06:
		t.TMA = v
	case 0xFF07:
		t.TAC = v & 7
	}
	if before && !t.signal() {
		m.timaInc()
	}
}

func (m *Machine) timerRead(a uint16) uint8 {
	t := &m.Timer
	switch a {
	case 0xFF04:
		return uint8(t.Div >> 8)
	case 0xFF05:
		return t.TIMA
	case 0xFF06:
		return t.TMA
	}
	return t.TAC | 0xF8
}
