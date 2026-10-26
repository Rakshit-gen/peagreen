package main

import (
	"image/png"
	"os"
	"path/filepath"
	"testing"
)

func testROM(t *testing.T, name string) string {
	t.Helper()
	p := filepath.Join("../../testdata/roms", name)
	if _, err := os.Stat(p); err != nil {
		t.Skip("test ROMs missing; run scripts/testroms.sh")
	}
	return p
}

func TestRunUntil(t *testing.T) {
	rom := testROM(t, "blargg/cpu_instrs/individual/06-ld r,r.gb")
	if err := run([]string{"-until", "Passed", "-frames", "600", rom}); err != nil {
		t.Fatal(err)
	}
	if err := run([]string{"-until", "nonsense", "-frames", "10", rom}); err == nil {
		t.Fatal("a string that never shows up should be an error")
	}
}

func TestRunPNGAndWAV(t *testing.T) {
	rom := testROM(t, "dmg-acid2.gb")
	dir := t.TempDir()
	shot, wav := filepath.Join(dir, "a.png"), filepath.Join(dir, "a.wav")
	if err := run([]string{"-frames", "30", "-png", shot, "-scale", "2", "-wav", wav, rom}); err != nil {
		t.Fatal(err)
	}
	f, err := os.Open(shot)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	img, err := png.Decode(f)
	if err != nil {
		t.Fatal(err)
	}
	if img.Bounds().Dx() != 320 {
		t.Fatalf("width %d, want 320", img.Bounds().Dx())
	}
	// Half a second of 16-bit stereo at 48 kHz, give or take a frame.
	fi, err := os.Stat(wav)
	if err != nil {
		t.Fatal(err)
	}
	if n := fi.Size() - 44; n < 90000 || n > 100000 {
		t.Fatalf("%d bytes of sound for 30 frames", n)
	}
}

func TestLoadRefusesColorOnly(t *testing.T) {
	rom := make([]byte, 0x8000)
	rom[0x143] = 0xC0
	path := filepath.Join(t.TempDir(), "color.gbc")
	if err := os.WriteFile(path, rom, 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := load(path); err == nil {
		t.Fatal("a Color-only game should be refused")
	}
	rom[0x143] = 0x80 // works on both
	os.WriteFile(path, rom, 0o644)
	if _, err := load(path); err != nil {
		t.Fatal(err)
	}
}
