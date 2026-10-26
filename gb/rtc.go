package gb

// RTC is the MBC3 real-time clock. It counts emulated seconds, so a run is
// the same every time; Advance catches it up on wall time between sessions.
type RTC struct {
	S, M, H uint8
	Days    uint16 // 9 bits
	Halt    bool
	Carry   bool // the day counter overflowed
	Sub     int  // cycles into the current second
	Latched [5]uint8
	Armed   bool // a 0 was written to the latch register
}

const cyclesPerSecond = 4194304

// tick adds emulated cycles.
func (r *RTC) tick(cycles int) {
	if r.Halt {
		return
	}
	r.Sub += cycles
	for r.Sub >= cyclesPerSecond {
		r.Sub -= cyclesPerSecond
		r.second()
	}
}

// Advance adds whole seconds, for time that passed while nothing ran.
func (r *RTC) Advance(seconds int64) {
	if r.Halt || seconds <= 0 {
		return
	}
	// Past a few days, skip whole 512-day cycles; the carry flag is all
	// that remembers them.
	const cycle = 512 * 86400
	if seconds > cycle {
		r.Carry = true
		seconds %= cycle
	}
	for range seconds {
		r.second()
	}
}

// second counts one second. The registers are 6, 6 and 5 bits wide, so a
// value written out of range counts up to the top and wraps to 0 without
// carrying, as the chip does.
func (r *RTC) second() {
	if r.S = (r.S + 1) & 63; r.S != 60 {
		return
	}
	r.S = 0
	if r.M = (r.M + 1) & 63; r.M != 60 {
		return
	}
	r.M = 0
	if r.H = (r.H + 1) & 31; r.H != 24 {
		return
	}
	r.H = 0
	r.Days++
	if r.Days == 512 {
		r.Days, r.Carry = 0, true
	}
}

func (r *RTC) dh() uint8 {
	v := uint8(r.Days>>8) & 1
	if r.Halt {
		v |= 0x40
	}
	if r.Carry {
		v |= 0x80
	}
	return v
}

// latch copies the live registers into the ones the game reads.
func (r *RTC) latch() {
	r.Latched = [5]uint8{r.S, r.M, r.H, uint8(r.Days), r.dh()}
}

// read returns latched register 0x08-0x0C.
func (r *RTC) read(reg int) uint8 {
	masks := [5]uint8{0x3F, 0x3F, 0x1F, 0xFF, 0xC1}
	return r.Latched[reg-8] & masks[reg-8]
}

// write sets live register 0x08-0x0C.
func (r *RTC) write(reg int, v uint8) {
	switch reg {
	case 8:
		r.S, r.Sub = v&63, 0
	case 9:
		r.M = v & 63
	case 10:
		r.H = v & 31
	case 11:
		r.Days = r.Days&0x100 | uint16(v)
	case 12:
		r.Days = r.Days&0xFF | uint16(v&1)<<8
		r.Halt = v&0x40 != 0
		r.Carry = v&0x80 != 0
	}
	// The latched copy follows writes, so a game reads back what it wrote.
	r.Latched[reg-8] = v
}
