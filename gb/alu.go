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
