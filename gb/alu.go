package gb

// alu runs operation y on A: ADD ADC SUB SBC AND XOR OR CP.
func (c *CPU) alu(y, v uint8) {
	var carry uint8
	if (y == 1 || y == 3) && c.flag(flagC) {
		carry = 1
	}
	switch y {
	case 0, 1:
		r := uint16(c.A) + uint16(v) + uint16(carry)
		c.setFlags(uint8(r) == 0, false, c.A&0xF+v&0xF+carry > 0xF, r > 0xFF)
		c.A = uint8(r)
	case 2, 3, 7:
		r := int(c.A) - int(v) - int(carry)
		c.setFlags(uint8(r) == 0, true, int(c.A&0xF)-int(v&0xF)-int(carry) < 0, r < 0)
		if y != 7 {
			c.A = uint8(r)
		}
	case 4:
		c.A &= v
		c.setFlags(c.A == 0, false, true, false)
	case 5:
		c.A ^= v
		c.setFlags(c.A == 0, false, false, false)
	case 6:
		c.A |= v
		c.setFlags(c.A == 0, false, false, false)
	}
}

// misc runs the eight one-byte instructions at 0x07-0x3F step 8: RLCA RRCA
// RLA RRA DAA CPL SCF CCF.
func (c *CPU) misc(y uint8) {
	a, cy := c.A, c.flag(flagC)
	switch y {
	case 0:
		c.A = a<<1 | a>>7
		c.setFlags(false, false, false, a&0x80 != 0)
	case 1:
		c.A = a>>1 | a<<7
		c.setFlags(false, false, false, a&1 != 0)
	case 2:
		c.A = a << 1
		if cy {
			c.A |= 1
		}
		c.setFlags(false, false, false, a&0x80 != 0)
	case 3:
		c.A = a >> 1
		if cy {
			c.A |= 0x80
		}
		c.setFlags(false, false, false, a&1 != 0)
	case 4:
		c.daa()
	case 5:
		c.A = ^a
		c.F |= flagN | flagH
	case 6:
		c.setFlags(c.flag(flagZ), false, false, true)
	case 7:
		c.setFlags(c.flag(flagZ), false, false, !cy)
	}
}

// daa fixes A up into binary-coded decimal after an add or subtract.
func (c *CPU) daa() {
	a, cy := c.A, c.flag(flagC)
	var adj uint8
	if c.flag(flagH) || (!c.flag(flagN) && a&0xF > 9) {
		adj |= 0x06
	}
	if cy || (!c.flag(flagN) && a > 0x99) {
		adj |= 0x60
		cy = true
	}
	if c.flag(flagN) {
		a -= adj
	} else {
		a += adj
	}
	c.A = a
	c.setFlags(a == 0, c.flag(flagN), false, cy)
}

// shift runs CB operation y on v: RLC RRC RL RR SLA SRA SWAP SRL.
func (c *CPU) shift(y, v uint8) uint8 {
	var r uint8
	out := false
	switch y {
	case 0:
		r, out = v<<1|v>>7, v&0x80 != 0
	case 1:
		r, out = v>>1|v<<7, v&1 != 0
	case 2:
		r, out = v<<1, v&0x80 != 0
		if c.flag(flagC) {
			r |= 1
		}
	case 3:
		r, out = v>>1, v&1 != 0
		if c.flag(flagC) {
			r |= 0x80
		}
	case 4:
		r, out = v<<1, v&0x80 != 0
	case 5:
		r, out = v>>1|v&0x80, v&1 != 0
	case 6:
		r = v<<4 | v>>4
	case 7:
		r, out = v>>1, v&1 != 0
	}
	c.setFlags(r == 0, false, false, out)
	return r
}

func (c *CPU) inc(v uint8) uint8 {
	r := v + 1
	c.setFlags(r == 0, false, v&0xF == 0xF, c.flag(flagC))
	return r
}

func (c *CPU) dec(v uint8) uint8 {
	r := v - 1
	c.setFlags(r == 0, true, v&0xF == 0, c.flag(flagC))
	return r
}
