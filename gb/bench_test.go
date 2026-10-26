package gb

import "testing"

// BenchmarkFrame runs whole frames of blargg's cpu_instrs, which keeps
// the CPU busy and the LCD on. Real hardware runs 59.7 a second.
func BenchmarkFrame(b *testing.B) {
	m := loadROM(b, "blargg/cpu_instrs/cpu_instrs.gb")
	m.EnableAudio(48000)
	b.ResetTimer()
	for b.Loop() {
		m.RunFrame()
		m.Audio()
	}
}
