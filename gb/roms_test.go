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
