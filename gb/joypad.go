package gb

// Button is one of the eight buttons.
type Button uint8

// Buttons, as bits of Joypad.Held.
const (
	Right Button = 1 << iota
	Left
	Up
	Down
	A
	B
	Select
	Start
)

// Joypad is the button matrix behind 0xFF00. The game picks a row (the
// d-pad or the buttons) with bits 4 and 5, and reads it back in bits 0-3,
// where 0 means held.
type Joypad struct {
	Select uint8
	Held   Button
}

// Press holds buttons down. Pressing one raises the joypad interrupt.
func (m *Machine) Press(b Button) {
	if b&^m.Joy.Held != 0 {
		m.IF |= IntJoypad
	}
	m.Joy.Held |= b
}

// Release lets buttons go.
func (m *Machine) Release(b Button) { m.Joy.Held &^= b }

// SetButtons holds exactly the buttons in b.
func (m *Machine) SetButtons(b Button) {
	m.Release(m.Joy.Held &^ b)
	m.Press(b)
}

func (j *Joypad) read() uint8 {
	v := 0xC0 | j.Select | 0x0F
	if j.Select&0x10 == 0 {
		v &^= uint8(j.Held) & 0x0F
	}
	if j.Select&0x20 == 0 {
		v &^= uint8(j.Held) >> 4
	}
	return v
}
