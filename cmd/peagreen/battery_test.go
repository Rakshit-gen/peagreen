package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/Rakshit-gen/peagreen/gb"
)

func TestBattery(t *testing.T) {
	if got := savPath("/games/zelda.gb"); got != "/games/zelda.sav" {
		t.Fatalf("savPath = %q", got)
	}
	rom := make([]byte, 0x8000)
	rom[0x147], rom[0x149] = 0x09, 2 // ROM+RAM+battery, 8 KiB
	dir := t.TempDir()
	romPath := filepath.Join(dir, "game.gb")
	m, err := gb.New(rom)
	if err != nil {
		t.Fatal(err)
	}
	m.Write(0xA010, 0x77)
	b := &battery{path: savPath(romPath)}
	if err := b.save(m); err != nil {
		t.Fatal(err)
	}
	fi, _ := os.Stat(b.path)
	m2, _ := gb.New(rom)
	if err := loadBattery(m2, romPath); err != nil {
		t.Fatal(err)
	}
	if got := m2.Read(0xA010); got != 0x77 {
		t.Fatalf("loaded %#x, want 0x77", got)
	}
	// Nothing changed, so no write.
	os.Chtimes(b.path, fi.ModTime().Add(-3600e9), fi.ModTime().Add(-3600e9))
	before, _ := os.Stat(b.path)
	b.save(m)
	after, _ := os.Stat(b.path)
	if !after.ModTime().Equal(before.ModTime()) {
		t.Fatal("an unchanged save was written again")
	}
}
