// Package audio writes the emulator's sound out: to WAV files, and to a
// player program for live sound.
package audio

import (
	"encoding/binary"
	"io"
	"math"
)

// WAV writes 16-bit stereo PCM. The header's sizes are filled in on Close.
type WAV struct {
	w    io.WriteSeeker
	rate int
	n    int // bytes of sample data
	buf  []byte
}

// NewWAV starts a WAV file at rate samples per second.
func NewWAV(w io.WriteSeeker, rate int) (*WAV, error) {
	v := &WAV{w: w, rate: rate}
	return v, v.header()
}

func (v *WAV) header() error {
	h := make([]byte, 44)
	copy(h[0:], "RIFF")
	binary.LittleEndian.PutUint32(h[4:], uint32(36+v.n))
	copy(h[8:], "WAVEfmt ")
	binary.LittleEndian.PutUint32(h[16:], 16)
	binary.LittleEndian.PutUint16(h[20:], 1) // PCM
	binary.LittleEndian.PutUint16(h[22:], 2) // stereo
	binary.LittleEndian.PutUint32(h[24:], uint32(v.rate))
	binary.LittleEndian.PutUint32(h[28:], uint32(v.rate*4))
	binary.LittleEndian.PutUint16(h[32:], 4)
	binary.LittleEndian.PutUint16(h[34:], 16)
	copy(h[36:], "data")
	binary.LittleEndian.PutUint32(h[40:], uint32(v.n))
	_, err := v.w.Write(h)
	return err
}

// Write adds interleaved left, right samples from -1 to 1.
func (v *WAV) Write(samples []float32) error {
	v.buf = v.buf[:0]
	for _, s := range samples {
		v.buf = binary.LittleEndian.AppendUint16(v.buf, uint16(pcm16(s)))
	}
	n, err := v.w.Write(v.buf)
	v.n += n
	return err
}

// Close fills in the sizes. It doesn't close the writer underneath.
func (v *WAV) Close() error {
	if _, err := v.w.Seek(0, io.SeekStart); err != nil {
		return err
	}
	if err := v.header(); err != nil {
		return err
	}
	_, err := v.w.Seek(0, io.SeekEnd)
	return err
}

func pcm16(s float32) int16 {
	return int16(math.Round(float64(max(-1, min(1, s))) * 32767))
}
