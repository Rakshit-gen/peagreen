package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/Rakshit-gen/peagreen/gb"
)

// savPath is where a ROM's battery save lives: next to it, ending in .sav,
// as most emulators keep it.
func savPath(rom string) string {
	return strings.TrimSuffix(rom, filepath.Ext(rom)) + ".sav"
}

// loadBattery puts the save next to the ROM into the cartridge, if there
// is one.
func loadBattery(m *gb.Machine, rom string) error {
	b, err := os.ReadFile(savPath(rom))
	if os.IsNotExist(err) || !m.Cart.Battery {
		return nil
	}
	if err != nil {
		return err
	}
	return m.Cart.LoadSave(b, time.Now())
}

// battery writes saves when the cartridge RAM changes.
type battery struct {
	path string
	last []byte
}

// save writes the save file if the data changed since the last write. It
// writes a temporary file and renames it, so a crash can't leave half a
// save behind.
func (b *battery) save(m *gb.Machine) error {
	data := m.Cart.SaveData(time.Now())
	if data == nil {
		return nil
	}
	// The clock's timestamp at the end changes every second; only RAM
	// changes are worth a write on their own.
	ram := data[:len(m.Cart.RAM)]
	if b.last != nil && bytes.Equal(ram, b.last) {
		return nil
	}
	tmp := b.path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return err
	}
	if err := os.Rename(tmp, b.path); err != nil {
		return err
	}
	b.last = append(b.last[:0], ram...)
	return nil
}
