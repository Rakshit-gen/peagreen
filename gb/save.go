package gb

import (
	"encoding/binary"
	"fmt"
	"time"
)

// SaveData is what a battery keeps: the cartridge RAM, then, for carts
// with a clock, 48 bytes of clock in the layout BGB and VBA-M use, so
// saves move between them. It returns nil for carts without a battery.
func (c *Cart) SaveData(now time.Time) []byte {
	if !c.Battery {
		return nil
	}
	out := append([]byte(nil), c.RAM...)
	if c.HasRTC {
		r := &c.RTC
		for _, v := range []uint8{r.S, r.M, r.H, uint8(r.Days), r.dh()} {
			out = binary.LittleEndian.AppendUint32(out, uint32(v))
		}
		for _, v := range r.Latched {
			out = binary.LittleEndian.AppendUint32(out, uint32(v))
		}
		out = binary.LittleEndian.AppendUint64(out, uint64(now.Unix()))
	}
	return out
}

// LoadSave puts saved data back, and runs the clock on by the time that
// passed since it was saved.
func (c *Cart) LoadSave(b []byte, now time.Time) error {
	if len(b) < len(c.RAM) {
		return fmt.Errorf("save is %d bytes, the cartridge has %d of RAM", len(b), len(c.RAM))
	}
	copy(c.RAM, b)
	rest := b[len(c.RAM):]
	if !c.HasRTC || len(rest) < 44 {
		return nil
	}
	u := func(i int) uint8 { return uint8(binary.LittleEndian.Uint32(rest[i*4:])) }
	r := &c.RTC
	r.S, r.M, r.H = u(0)&63, u(1)&63, u(2)&31
	r.Days = uint16(u(3)) | uint16(u(4)&1)<<8
	r.Halt, r.Carry = u(4)&0x40 != 0, u(4)&0x80 != 0
	for i := range r.Latched {
		r.Latched[i] = u(5 + i)
	}
	if len(rest) >= 48 {
		then := int64(binary.LittleEndian.Uint64(rest[40:]))
		r.Advance(now.Unix() - then)
	}
	return nil
}
