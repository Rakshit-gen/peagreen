package gb

// Channel is one sound channel. The four channels share the struct; each
// uses the fields that apply to it.
type Channel struct {
	On     bool
	DAC    bool
	Length int // counts down to silence when LenOn
	LenOn  bool
	Freq   uint16
	Timer  int // T-cycles to the next waveform step

	// Volume envelope (1, 2 and 4).
	Vol, EnvPeriod, EnvTimer uint8
	EnvUp                    bool

	// Square duty (1 and 2) and the step within it; wave position (3).
	Duty, Pos uint8

	// Frequency sweep (1).
	SweepPeriod, SweepTimer, SweepShift uint8
	SweepDown, SweepOn                  bool
	Negated                             bool // a subtracting calculation ran since trigger
	Shadow                              uint16

	// Wave volume shift (3) and the sample last read.
	WaveShift uint8
	Sample    uint8

	// Noise (4).
	LFSR   uint16
	Narrow bool // 7-bit LFSR
	NDiv   uint8
	NShift uint8
}

// APU is the sound unit.
type APU struct {
	On   bool
	Regs [0x30]uint8 // 0xFF10-0xFF3F as written, wave RAM at the end
	Ch   [4]Channel
	Seq  uint8 // frame sequencer step
	Div  bool  // DIV bit 12 last M-cycle, whose falling edge steps Seq

	// WaveAt is the T-cycle channel 3 last read wave RAM. Only right then
	// can the CPU reach wave RAM while the channel plays, and then only the
	// byte the channel is on. Now is the T-cycle of the current CPU access.
	WaveAt uint64
	Now    uint64

	rate     int // output samples per second, 0 for none
	frac     int
	sumL     float64
	sumR     float64
	n        int
	capL     float64
	capR     float64
	highPass float64
	samples  []float32
}

// readMask is what each register reads back as besides what was written:
// write-only bits and unused registers read as 1.
var readMask = [0x30]uint8{
	0x80, 0x3F, 0x00, 0xFF, 0xBF, // NR10-NR14
	0xFF, 0x3F, 0x00, 0xFF, 0xBF, // NR20-NR24
	0x7F, 0xFF, 0x9F, 0xFF, 0xBF, // NR30-NR34
	0xFF, 0xFF, 0x00, 0x00, 0xBF, // NR40-NR44
	0x00, 0x00, 0x70, // NR50-NR52
	0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF,
}

func (a *APU) read(addr uint16) uint8 {
	i := addr - 0xFF10
	if i >= 0x20 {
		if c := &a.Ch[2]; c.On {
			if !a.waveOpen() {
				return 0xFF
			}
			return a.Regs[0x20+c.Pos/2]
		}
		return a.Regs[i]
	}
	if addr == 0xFF26 {
		v := uint8(0x70)
		if a.On {
			v |= 0x80
		}
		for n := range a.Ch {
			if a.Ch[n].On {
				v |= 1 << n
			}
		}
		return v
	}
	return a.Regs[i] | readMask[i]
}

func (a *APU) write(addr uint16, v uint8) {
	i := addr - 0xFF10
	if i >= 0x20 {
		if c := &a.Ch[2]; c.On {
			if a.waveOpen() {
				a.Regs[0x20+c.Pos/2] = v
			}
			return
		}
		a.Regs[i] = v
		return
	}
	if addr == 0xFF26 {
		on := v&0x80 != 0
		if a.On && !on {
			// Powering off clears every register, but on the DMG the
			// length counters survive.
			clear(a.Regs[:0x16])
			for n := range a.Ch {
				a.Ch[n] = Channel{Length: a.Ch[n].Length}
			}
		}
		if !a.On && on {
			a.Seq = 0
		}
		a.On = on
		return
	}
	if !a.On {
		// The DMG still takes length writes while off.
		switch i {
		case 0x01, 0x06, 0x10:
			a.Ch[i/5].Length = 64 - int(v&63)
		case 0x0B:
			a.Ch[2].Length = 256 - int(v)
		}
		return
	}
	a.Regs[i] = v
	n := int(i) / 5
	if i >= 0x14 {
		return // NR50, NR51 are read straight from Regs
	}
	c := &a.Ch[n]
	switch i % 5 {
	case 0:
		switch n {
		case 0:
			c.SweepPeriod, c.SweepDown, c.SweepShift = v>>4&7, v&8 != 0, v&7
			// Leaving subtract mode after it has been used cuts the channel.
			if c.Negated && !c.SweepDown {
				c.On = false
			}
		case 2:
			c.DAC = v&0x80 != 0
			if !c.DAC {
				c.On = false
			}
		}
	case 1:
		switch n {
		case 0, 1:
			c.Duty = v >> 6
			c.Length = 64 - int(v&63)
		case 2:
			c.Length = 256 - int(v)
		case 3:
			c.Length = 64 - int(v&63)
		}
	case 2:
		switch n {
		case 2:
			c.WaveShift = v >> 5 & 3
		default:
			c.DAC = v&0xF8 != 0
			if !c.DAC {
				c.On = false
			}
		}
	case 3:
		switch n {
		case 3:
			c.NShift, c.Narrow, c.NDiv = v>>4, v&8 != 0, v&7
		default:
			c.Freq = c.Freq&0x700 | uint16(v)
		}
	case 4:
		if n != 3 {
			c.Freq = c.Freq&0xFF | uint16(v&7)<<8
		}
		// In the first half of a length period (the next sequencer step
		// won't clock length), turning length on clocks it once at once,
		// and a trigger that reloads it to the maximum takes one off.
		firstHalf := a.Seq&1 == 1
		was := c.LenOn
		c.LenOn = v&0x40 != 0
		if !was && c.LenOn && firstHalf && c.Length > 0 {
			if c.Length--; c.Length == 0 && v&0x80 == 0 {
				c.On = false
			}
		}
		if v&0x80 != 0 {
			if n == 2 {
				a.corruptWave()
			}
			if c.Length == 0 {
				c.Length = lengthMax(n)
				if c.LenOn && firstHalf {
					c.Length--
				}
			}
			a.trigger(n)
		}
	}
}

func (a *APU) waveOpen() bool { return a.Now-a.WaveAt <= 1 }

// corruptWave is the DMG bug where retriggering channel 3 just as it
// reads a sample overwrites the start of wave RAM with the block it was
// about to read, when that read was due in the next 2 T-cycles.
func (a *APU) corruptWave() {
	c := &a.Ch[2]
	if !c.On || c.Timer < 1 || c.Timer > 2 {
		return
	}
	b := int((c.Pos+1)&31) / 2
	w := a.Regs[0x20:0x30]
	if b < 4 {
		w[0] = w[b]
	} else {
		copy(w[0:4], w[b&^3:b&^3+4])
	}
}

func lengthMax(n int) int {
	if n == 2 {
		return 256
	}
	return 64
}

// trigger restarts channel n.
func (a *APU) trigger(n int) {
	c := &a.Ch[n]
	c.On = c.DAC
	if c.Length == 0 {
		c.Length = lengthMax(n)
	}
	c.Timer = a.period(n)
	nrx2 := a.Regs[n*5+2]
	c.Vol, c.EnvUp, c.EnvPeriod = nrx2>>4, nrx2&8 != 0, nrx2&7
	c.EnvTimer = c.EnvPeriod
	switch n {
	case 0:
		c.Shadow = c.Freq
		c.Negated = false
		c.SweepTimer = c.SweepPeriod
		if c.SweepTimer == 0 {
			c.SweepTimer = 8
		}
		c.SweepOn = c.SweepPeriod != 0 || c.SweepShift != 0
		if c.SweepShift != 0 {
			a.sweepCalc()
		}
	case 2:
		c.Pos = 0
		// The first sample comes a few cycles late.
		c.Timer += 6
	case 3:
		c.LFSR = 0x7FFF
	}
}

var noiseDivisors = [8]int{8, 16, 32, 48, 64, 80, 96, 112}

// period is T-cycles per waveform step of channel n.
func (a *APU) period(n int) int {
	c := &a.Ch[n]
	switch n {
	case 2:
		return (2048 - int(c.Freq)) * 2
	case 3:
		return noiseDivisors[c.NDiv] << c.NShift
	}
	return (2048 - int(c.Freq)) * 4
}

// sweepCalc works out channel 1's next frequency and silences it if that
// overflows.
func (a *APU) sweepCalc() uint16 {
	c := &a.Ch[0]
	d := c.Shadow >> c.SweepShift
	f := c.Shadow + d
	if c.SweepDown {
		f = c.Shadow - d
		c.Negated = true
	}
	if f > 2047 {
		c.On = false
	}
	return f
}

func (m *Machine) apuTick() {
	a := &m.APU
	bit := m.Timer.Div&(1<<12) != 0
	if a.Div && !bit && a.On {
		a.step()
	}
	a.Div = bit
	a.Now = m.Cycles
	if !a.On {
		a.output()
		return
	}
	for n := range a.Ch {
		c := &a.Ch[n]
		c.Timer -= 4
		for c.Timer <= 0 {
			at := m.Cycles + uint64(c.Timer) // Timer is <= 0 here
			c.Timer += a.period(n)
			switch n {
			case 0, 1:
				c.Pos = (c.Pos + 1) & 7
			case 2:
				c.Pos = (c.Pos + 1) & 31
				a.WaveAt = at
				b := a.Regs[0x20+c.Pos/2]
				if c.Pos&1 == 0 {
					b >>= 4
				}
				c.Sample = b & 0x0F
			case 3:
				x := (c.LFSR ^ c.LFSR>>1) & 1
				c.LFSR = c.LFSR>>1 | x<<14
				if c.Narrow {
					c.LFSR = c.LFSR&^0x40 | x<<6
				}
			}
		}
	}
	a.output()
}

// step runs the 512 Hz frame sequencer: length at 256 Hz, sweep at 128 Hz
// and the envelopes at 64 Hz.
func (a *APU) step() {
	if a.Seq&1 == 0 {
		for n := range a.Ch {
			c := &a.Ch[n]
			if c.LenOn && c.Length > 0 {
				if c.Length--; c.Length == 0 {
					c.On = false
				}
			}
		}
	}
	if a.Seq == 2 || a.Seq == 6 {
		a.sweep()
	}
	if a.Seq == 7 {
		for _, n := range []int{0, 1, 3} {
			c := &a.Ch[n]
			if c.EnvPeriod == 0 {
				continue
			}
			if c.EnvTimer--; c.EnvTimer > 0 {
				continue
			}
			c.EnvTimer = c.EnvPeriod
			if c.EnvUp && c.Vol < 15 {
				c.Vol++
			} else if !c.EnvUp && c.Vol > 0 {
				c.Vol--
			}
		}
	}
	a.Seq = (a.Seq + 1) & 7
}

func (a *APU) sweep() {
	c := &a.Ch[0]
	if c.SweepTimer--; c.SweepTimer > 0 {
		return
	}
	c.SweepTimer = c.SweepPeriod
	if c.SweepTimer == 0 {
		c.SweepTimer = 8
	}
	if !c.SweepOn || c.SweepPeriod == 0 {
		return
	}
	f := a.sweepCalc()
	if f <= 2047 && c.SweepShift != 0 {
		c.Freq, c.Shadow = f, f
		a.sweepCalc()
	}
}

var duties = [4]uint8{0b00000001, 0b10000001, 0b10000111, 0b01111110}

// level is channel n's output, 0-15.
func (a *APU) level(n int) uint8 {
	c := &a.Ch[n]
	if !c.On {
		return 0
	}
	switch n {
	case 2:
		if c.WaveShift == 0 {
			return 0
		}
		return c.Sample >> (c.WaveShift - 1)
	case 3:
		if c.LFSR&1 != 0 {
			return 0
		}
		return c.Vol
	}
	if duties[c.Duty]>>(7-c.Pos)&1 == 0 {
		return 0
	}
	return c.Vol
}

// output mixes the channels into the sample stream.
func (a *APU) output() {}
