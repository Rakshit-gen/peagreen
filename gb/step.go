package gb

// Every memory access costs one M-cycle, and the rest of the machine moves
// on before it happens. Instruction timing comes out of the accesses plus
// the internal delays each instruction spends.

func (m *Machine) rd(a uint16) uint8 {
	m.tick()
	if m.blocked(a) {
		return 0xFF
	}
	return m.Read(a)
}

func (m *Machine) wr(a uint16, v uint8) {
	m.tick()
	if !m.blocked(a) {
		m.Write(a, v)
	}
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

// rp reads 16-bit register pair i: BC DE HL SP.
func (m *Machine) rp(i uint8) uint16 {
	c := &m.CPU
	switch i {
	case 0:
		return c.bc()
	case 1:
		return c.de()
	case 2:
		return c.hl()
	}
	return c.SP
}

func (m *Machine) setRP(i uint8, v uint16) {
	c := &m.CPU
	switch i {
	case 0:
		c.setBC(v)
	case 1:
		c.setDE(v)
	case 2:
		c.setHL(v)
	default:
		c.SP = v
	}
}

func (m *Machine) push(v uint16) {
	c := &m.CPU
	c.SP--
	m.wr(c.SP, uint8(v>>8))
	c.SP--
	m.wr(c.SP, uint8(v))
}

func (m *Machine) pop() uint16 {
	c := &m.CPU
	lo := m.rd(c.SP)
	c.SP++
	hi := m.rd(c.SP)
	c.SP++
	return uint16(hi)<<8 | uint16(lo)
}

// indirect is LD (rr),A and LD A,(rr) for rr = BC, DE, HL+ and HL-.
func (m *Machine) indirect(y uint8) {
	c := &m.CPU
	var a uint16
	switch y >> 1 {
	case 0:
		a = c.bc()
	case 1:
		a = c.de()
	case 2:
		a = c.hl()
		c.setHL(a + 1)
	case 3:
		a = c.hl()
		c.setHL(a - 1)
	}
	if y&1 == 0 {
		m.wr(a, c.A)
	} else {
		c.A = m.rd(a)
	}
}

// Step runs one instruction, or one M-cycle of a halted or hung CPU.
func (m *Machine) Step() {
	c := &m.CPU
	if m.interrupt() {
		return
	}
	if c.Locked || c.Halted {
		m.tick()
		return
	}
	ei := c.EIPending
	m.exec(m.fetch())
	// EI lands after the instruction that follows it, unless that was DI.
	if ei && c.EIPending {
		c.IME, c.EIPending = true, false
	}
}

// interrupt wakes a halted CPU when an enabled interrupt is pending, and
// jumps to its handler if IME is set. The dispatch takes five M-cycles.
func (m *Machine) interrupt() bool {
	c := &m.CPU
	if c.Locked || m.IE&m.IF&0x1F == 0 {
		return false
	}
	c.Halted = false
	if !c.IME {
		return false
	}
	c.IME = false
	m.tick()
	m.tick()
	c.SP--
	m.wr(c.SP, uint8(c.PC>>8))
	// Pushing the high byte can land on IE and cancel the interrupt, in
	// which case the CPU ends up at 0x0000.
	pending := m.IE & m.IF & 0x1F
	c.SP--
	m.wr(c.SP, uint8(c.PC))
	c.PC = 0
	for i := range 5 {
		if pending&(1<<i) != 0 {
			m.IF &^= 1 << i
			c.PC = 0x40 + uint16(i)*8
			break
		}
	}
	m.tick()
	return true
}

func (m *Machine) exec(op uint8) {
	c := &m.CPU
	x, y, z := op>>6, op>>3&7, op&7
	switch {
	case op == 0x00: // NOP
	case op == 0x76: // HALT
		// With interrupts off and one already pending, HALT doesn't halt,
		// and the CPU reads the next byte twice.
		if !c.IME && m.IE&m.IF&0x1F != 0 {
			c.HaltBug = true
		} else {
			c.Halted = true
		}
	case op == 0x10: // STOP
		// A DMG with no button held stays stopped until one is pressed;
		// nothing commercial relies on that, so this only resets DIV.
		m.fetch()
		m.stop()
	case op == 0xF3: // DI
		c.IME, c.EIPending = false, false
	case op == 0xFB: // EI
		c.EIPending = true
	case op == 0xD9: // RETI
		c.PC = m.pop()
		m.tick()
		c.IME = true
	case x == 1: // LD r,r
		m.setReg(y, m.reg(z))
	case x == 0 && z == 6: // LD r,n
		m.setReg(y, m.fetch())
	case x == 0 && z == 1 && y&1 == 0: // LD rr,nn
		m.setRP(y>>1, m.fetch16())
	case x == 0 && z == 2:
		m.indirect(y)
	case op == 0x08: // LD (nn),SP
		a := m.fetch16()
		m.wr(a, uint8(c.SP))
		m.wr(a+1, uint8(c.SP>>8))
	case op == 0xE0: // LDH (n),A
		m.wr(0xFF00|uint16(m.fetch()), c.A)
	case op == 0xF0: // LDH A,(n)
		c.A = m.rd(0xFF00 | uint16(m.fetch()))
	case op == 0xE2: // LD (C),A
		m.wr(0xFF00|uint16(c.C), c.A)
	case op == 0xF2: // LD A,(C)
		c.A = m.rd(0xFF00 | uint16(c.C))
	case op == 0xEA: // LD (nn),A
		m.wr(m.fetch16(), c.A)
	case op == 0xFA: // LD A,(nn)
		c.A = m.rd(m.fetch16())
	case op == 0xF9: // LD SP,HL
		m.tick()
		c.SP = c.hl()
	case x == 3 && z == 1 && y&1 == 0: // POP
		if v := m.pop(); y>>1 == 3 {
			c.setAF(v)
		} else {
			m.setRP(y>>1, v)
		}
	case x == 3 && z == 5 && y&1 == 0: // PUSH
		m.tick()
		if y>>1 == 3 {
			m.push(c.af())
		} else {
			m.push(m.rp(y >> 1))
		}
	case x == 2: // ALU A,r
		c.alu(y, m.reg(z))
	case x == 3 && z == 6: // ALU A,n
		c.alu(y, m.fetch())
	case x == 0 && z == 4: // INC r
		m.setReg(y, c.inc(m.reg(y)))
	case x == 0 && z == 5: // DEC r
		m.setReg(y, c.dec(m.reg(y)))
	case x == 0 && z == 3: // INC rr, DEC rr
		m.tick()
		if y&1 == 0 {
			m.setRP(y>>1, m.rp(y>>1)+1)
		} else {
			m.setRP(y>>1, m.rp(y>>1)-1)
		}
	case x == 0 && z == 1: // ADD HL,rr
		m.tick()
		hl, v := c.hl(), m.rp(y>>1)
		c.setFlags(c.flag(flagZ), false, hl&0xFFF+v&0xFFF > 0xFFF, uint32(hl)+uint32(v) > 0xFFFF)
		c.setHL(hl + v)
	case x == 0 && z == 7:
		c.misc(y)
	case op == 0x18 || (x == 0 && z == 0 && y >= 4): // JR e, JR cc,e
		e := int8(m.fetch())
		if op == 0x18 || c.cond(y-4) {
			m.tick()
			c.PC += uint16(e)
		}
	case op == 0xC3 || (x == 3 && z == 2 && y < 4): // JP nn, JP cc,nn
		a := m.fetch16()
		if op == 0xC3 || c.cond(y) {
			m.tick()
			c.PC = a
		}
	case op == 0xE9: // JP HL
		c.PC = c.hl()
	case op == 0xCD || (x == 3 && z == 4 && y < 4): // CALL nn, CALL cc,nn
		a := m.fetch16()
		if op == 0xCD || c.cond(y) {
			m.tick()
			m.push(c.PC)
			c.PC = a
		}
	case op == 0xC9: // RET
		c.PC = m.pop()
		m.tick()
	case x == 3 && z == 0 && y < 4: // RET cc
		m.tick()
		if c.cond(y) {
			c.PC = m.pop()
			m.tick()
		}
	case x == 3 && z == 7: // RST
		m.tick()
		m.push(c.PC)
		c.PC = uint16(y) * 8
	case op == 0xCB:
		m.execCB(m.fetch())
	case op == 0xE8: // ADD SP,e
		v := c.spPlus(m.fetch())
		m.tick()
		m.tick()
		c.SP = v
	case op == 0xF8: // LD HL,SP+e
		v := c.spPlus(m.fetch())
		m.tick()
		c.setHL(v)
	default:
		c.Locked = true
	}
}

// execCB runs the second byte of a CB-prefixed instruction.
func (m *Machine) execCB(op uint8) {
	c := &m.CPU
	x, y, z := op>>6, op>>3&7, op&7
	switch x {
	case 0:
		m.setReg(z, c.shift(y, m.reg(z)))
	case 1: // BIT
		v := m.reg(z)
		c.setFlags(v&(1<<y) == 0, false, true, c.flag(flagC))
	case 2: // RES
		m.setReg(z, m.reg(z)&^(1<<y))
	case 3: // SET
		m.setReg(z, m.reg(z)|1<<y)
	}
}

// cond tests condition i: NZ Z NC C.
func (c *CPU) cond(i uint8) bool {
	switch i {
	case 0:
		return !c.flag(flagZ)
	case 1:
		return c.flag(flagZ)
	case 2:
		return !c.flag(flagC)
	}
	return c.flag(flagC)
}

// spPlus adds a signed byte to SP. The flags come from adding the low byte
// as if it were unsigned.
func (c *CPU) spPlus(e uint8) uint16 {
	sp := c.SP
	c.setFlags(false, false, sp&0xF+uint16(e&0xF) > 0xF, sp&0xFF+uint16(e) > 0xFF)
	return sp + uint16(int8(e))
}
