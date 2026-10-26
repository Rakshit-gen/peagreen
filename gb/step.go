package gb

// Every memory access costs one M-cycle, and the rest of the machine moves
// on before it happens. Instruction timing comes out of the accesses plus
// the internal delays each instruction spends.

func (m *Machine) rd(a uint16) uint8 {
	m.tick()
	return m.Read(a)
}

func (m *Machine) wr(a uint16, v uint8) {
	m.tick()
	m.Write(a, v)
}

func (m *Machine) fetch() uint8 {
	v := m.rd(m.CPU.PC)
	if m.CPU.HaltBug {
		m.CPU.HaltBug = false
	} else {
		m.CPU.PC++
	}
	return v
}

func (m *Machine) fetch16() uint16 {
	lo := m.fetch()
	return uint16(m.fetch())<<8 | uint16(lo)
}

// reg reads operand register i in opcode order: B C D E H L (HL) A.
func (m *Machine) reg(i uint8) uint8 {
	c := &m.CPU
	switch i {
	case 0:
		return c.B
	case 1:
		return c.C
	case 2:
		return c.D
	case 3:
		return c.E
	case 4:
		return c.H
	case 5:
		return c.L
	case 6:
		return m.rd(c.hl())
	}
	return c.A
}

func (m *Machine) setReg(i, v uint8) {
	c := &m.CPU
	switch i {
	case 0:
		c.B = v
	case 1:
		c.C = v
	case 2:
		c.D = v
	case 3:
		c.E = v
	case 4:
		c.H = v
	case 5:
		c.L = v
	case 6:
		m.wr(c.hl(), v)
	default:
		c.A = v
	}
}

// Step runs one instruction, or one M-cycle of a halted or hung CPU.
func (m *Machine) Step() {
	c := &m.CPU
	if c.Locked || c.Halted {
		m.tick()
		return
	}
	m.exec(m.fetch())
}

func (m *Machine) exec(op uint8) {
	c := &m.CPU
	x, y, z := op>>6, op>>3&7, op&7
	switch {
	case op == 0x00: // NOP
	case op == 0x76: // HALT
		c.Halted = true
	case x == 1: // LD r,r
		m.setReg(y, m.reg(z))
	case x == 0 && z == 6: // LD r,n
		m.setReg(y, m.fetch())
	default:
		c.Locked = true
	}
}
