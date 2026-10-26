package gb

import (
	"bytes"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// romDir is where scripts/testroms.sh puts the public test ROMs.
const romDir = "../testdata/roms"

func loadROM(t *testing.T, path string) *Machine {
	t.Helper()
	rom, err := os.ReadFile(filepath.Join(romDir, path))
	if os.IsNotExist(err) {
		t.Skip("test ROMs missing; run scripts/testroms.sh")
	}
	if err != nil {
		t.Fatal(err)
	}
	m, err := New(rom)
	if err != nil {
		t.Fatal(err)
	}
	return m
}

// runSerial runs until the serial output says Passed or Failed, or the
// emulated time runs out.
func runSerial(m *Machine, seconds int) string {
	var out bytes.Buffer
	m.SetSerial(&out)
	limit := m.Cycles + uint64(seconds)*cyclesPerSecond
	for m.Cycles < limit {
		for range 10000 {
			m.Step()
		}
		if s := out.String(); strings.Contains(s, "Passed") || strings.Contains(s, "Failed") {
			break
		}
	}
	return out.String()
}

func TestBlarggCPU(t *testing.T) {
	for _, name := range []string{
		"01-special", "02-interrupts", "03-op sp,hl", "04-op r,imm", "05-op rp",
		"06-ld r,r", "07-jr,jp,call,ret,rst", "08-misc instrs", "09-op r,r",
		"10-bit ops", "11-op a,(hl)",
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			m := loadROM(t, "blargg/cpu_instrs/individual/"+name+".gb")
			if out := runSerial(m, 60); !strings.Contains(out, "Passed") {
				t.Fatalf("serial said:\n%s", out)
			}
		})
	}
}

func TestBlarggTiming(t *testing.T) {
	for _, name := range []string{
		"instr_timing/instr_timing",
		"mem_timing/individual/01-read_timing",
		"mem_timing/individual/02-write_timing",
		"mem_timing/individual/03-modify_timing",
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			m := loadROM(t, "blargg/"+name+".gb")
			if out := runSerial(m, 60); !strings.Contains(out, "Passed") {
				t.Fatalf("serial said:\n%s", out)
			}
		})
	}
}

// TestAcid2 compares the screen after dmg-acid2 has drawn its face with
// the reference picture from the test's author.
func TestAcid2(t *testing.T) {
	m := loadROM(t, "dmg-acid2.gb")
	f, err := os.Open(filepath.Join(romDir, "dmg-acid2.png"))
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	ref, err := png.Decode(f)
	if err != nil {
		t.Fatal(err)
	}
	for range 60 {
		m.RunFrame()
	}
	bad := 0
	for y := range Height {
		for x := range Width {
			g := color.GrayModel.Convert(ref.At(x, y)).(color.Gray).Y
			want := uint8(3 - (int(g)+42)/85)
			if m.PPU.Screen[y][x] != want {
				if bad < 10 {
					t.Errorf("(%d,%d): shade %d, want %d", x, y, m.PPU.Screen[y][x], want)
				}
				bad++
			}
		}
	}
	if bad > 0 {
		t.Fatalf("%d pixels differ", bad)
	}
}

// mooneye reports through registers: after LD B,B, a pass leaves the
// Fibonacci numbers 3 5 8 13 21 34 in B-L, and a failure leaves 0x42.
func runMooneye(m *Machine, seconds int) (passed, done bool) {
	limit := m.Cycles + uint64(seconds)*cyclesPerSecond
	for m.Cycles < limit {
		m.Step()
		c := &m.CPU
		if c.B == 3 && c.C == 5 && c.D == 8 && c.E == 13 && c.H == 21 && c.L == 34 {
			return true, true
		}
		if c.B == 0x42 && c.C == 0x42 && c.D == 0x42 && c.E == 0x42 && c.H == 0x42 && c.L == 0x42 {
			return false, true
		}
	}
	return false, false
}

// mooneyePass lists the acceptance tests that pass. Tests for other models
// (dmg0, mgb, sgb, cgb) are left out: this is a DMG ABC.
var mooneyePass = []string{
	"bits/mem_oam", "bits/reg_f", "bits/unused_hwio-GS", "boot_regs-dmgABC",
	"di_timing-GS", "div_timing", "ei_sequence", "ei_timing", "halt_ime0_ei",
	"halt_ime0_nointr_timing", "halt_ime1_timing", "halt_ime1_timing2-GS",
	"if_ie_registers", "instr/daa", "interrupts/ie_push", "intr_timing",
	"oam_dma/basic", "oam_dma/reg_read", "oam_dma/sources-GS", "pop_timing",
	"ppu/intr_1_2_timing-GS", "ppu/intr_2_0_timing", "ppu/stat_irq_blocking",
	"rapid_di_ei", "reti_intr_timing", "timer/div_write", "timer/rapid_toggle",
	"timer/tim00", "timer/tim00_div_trigger", "timer/tim01", "timer/tim01_div_trigger",
	"timer/tim10", "timer/tim10_div_trigger", "timer/tim11", "timer/tim11_div_trigger",
	// These time instructions by racing them against OAM DMA.
	"add_sp_e_timing", "call_cc_timing", "call_cc_timing2", "call_timing",
	"call_timing2", "jp_cc_timing", "jp_timing", "ld_hl_sp_e_timing",
	"oam_dma_restart", "oam_dma_timing", "push_timing", "ret_cc_timing",
	"ret_timing", "reti_timing", "rst_timing",
	"timer/tima_reload", "timer/tima_write_reloading", "timer/tma_write_reloading",
}

func TestMooneye(t *testing.T) {
	for _, name := range mooneyePass {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			m := loadROM(t, "mooneye/acceptance/"+name+".gb")
			passed, done := runMooneye(m, 10)
			if !done {
				t.Fatal("never finished")
			}
			if !passed {
				t.Fatalf("failed: A=%02X B=%02X C=%02X D=%02X E=%02X H=%02X L=%02X",
					m.CPU.A, m.CPU.B, m.CPU.C, m.CPU.D, m.CPU.E, m.CPU.H, m.CPU.L)
			}
		})
	}
}

func TestMooneyeMappers(t *testing.T) {
	for _, name := range []string{
		"mbc1/bits_bank1", "mbc1/bits_bank2", "mbc1/bits_mode", "mbc1/bits_ramg",
		"mbc1/ram_256kb", "mbc1/ram_64kb", "mbc1/rom_16Mb", "mbc1/rom_1Mb",
		"mbc1/rom_2Mb", "mbc1/rom_4Mb", "mbc1/rom_512kb", "mbc1/rom_8Mb",
		"mbc1/multicart_rom_8Mb",
		"mbc2/bits_ramg", "mbc2/bits_romb", "mbc2/bits_unused", "mbc2/ram",
		"mbc2/rom_1Mb", "mbc2/rom_2Mb", "mbc2/rom_512kb",
		"mbc5/rom_16Mb", "mbc5/rom_1Mb", "mbc5/rom_2Mb", "mbc5/rom_32Mb",
		"mbc5/rom_4Mb", "mbc5/rom_512kb", "mbc5/rom_64Mb", "mbc5/rom_8Mb",
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			m := loadROM(t, "mooneye/emulator-only/"+name+".gb")
			if passed, done := runMooneye(m, 10); !passed {
				t.Fatalf("passed=%v finished=%v", passed, done)
			}
		})
	}
}
