package main

import (
	"errors"
	"fmt"
	"os"

	"github.com/Rakshit-gen/peagreen/gb"
)

func info(args []string) error {
	if len(args) != 1 {
		return errors.New("usage: peagreen info game.gb")
	}
	rom, err := os.ReadFile(args[0])
	if err != nil {
		return err
	}
	h, err := gb.ParseHeader(rom)
	if err != nil {
		return err
	}
	ok := func(b bool) string {
		if b {
			return "ok"
		}
		return "wrong"
	}
	fmt.Printf("title      %s\n", h.Title)
	fmt.Printf("cartridge  %s\n", gb.TypeName(h.Type))
	fmt.Printf("rom        %d KiB in %d banks (file is %d KiB)\n", h.ROMBanks*16, h.ROMBanks, len(rom)/1024)
	if h.RAMSize > 0 {
		fmt.Printf("ram        %d KiB\n", h.RAMSize/1024)
	}
	switch {
	case h.CGB == 0xC0:
		fmt.Println("color      Game Boy Color only; it won't run here")
	case h.CGB&0x80 != 0:
		fmt.Println("color      enhanced for Game Boy Color, runs in original mode")
	}
	fmt.Printf("header     checksum %s\n", ok(gb.HeaderOK(rom)))
	fmt.Printf("global     checksum %s (nothing checks this one)\n", ok(gb.GlobalOK(rom)))
	if _, err := gb.NewCart(rom); err != nil {
		fmt.Printf("supported  no: %v\n", err)
	}
	return nil
}
