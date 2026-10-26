package main

import (
	"bytes"
	"errors"
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/Rakshit-gen/peagreen/audio"
	"github.com/Rakshit-gen/peagreen/gb"
	"github.com/Rakshit-gen/peagreen/screen"
)

func run(args []string) error {
	fs := flag.NewFlagSet("run", flag.ContinueOnError)
	frames := fs.Int("frames", 600, "frames to run (60 is about a second)")
	shot := fs.String("png", "", "write the last frame to this PNG")
	scale := fs.Int("scale", 1, "PNG scale")
	palName := fs.String("palette", "green", "PNG palette: green, pocket, gray or amber")
	serial := fs.Bool("serial", false, "print what the game sends over the link port")
	until := fs.String("until", "", "stop once the link port output contains this")
	presses := fs.String("press", "", `buttons to press, like "start@60,a@120+10"`)
	wavPath := fs.String("wav", "", "record the sound to this WAV file")
	fs.Usage = func() {
		fmt.Fprintln(fs.Output(), "usage: peagreen run [flags] game.gb")
		fs.PrintDefaults()
	}
	if err := fs.Parse(args); err != nil {
		return err
	}
	if fs.NArg() != 1 {
		fs.Usage()
		return errors.New("run takes one ROM")
	}
	pal, err := screen.Find(*palName)
	if err != nil {
		return err
	}
	script, err := parseScript(*presses)
	if err != nil {
		return err
	}
	m, err := load(fs.Arg(0))
	if err != nil {
		return err
	}
	var out bytes.Buffer
	if *serial || *until != "" {
		m.SetSerial(&out)
	}
	var wav *audio.WAV
	if *wavPath != "" {
		f, err := os.Create(*wavPath)
		if err != nil {
			return err
		}
		defer f.Close()
		m.EnableAudio(48000)
		if wav, err = audio.NewWAV(f, 48000); err != nil {
			return err
		}
	}
	printed := 0
	found := false
	for frame := 0; frame < *frames && !found; frame++ {
		m.SetButtons(held(script, frame))
		m.RunFrame()
		if wav != nil {
			if err := wav.Write(m.Audio()); err != nil {
				return err
			}
		}
		if *serial {
			os.Stdout.Write(out.Bytes()[printed:])
			printed = out.Len()
		}
		found = *until != "" && strings.Contains(out.String(), *until)
	}
	if wav != nil {
		if err := wav.Close(); err != nil {
			return err
		}
	}
	if *shot != "" {
		f, err := os.Create(*shot)
		if err != nil {
			return err
		}
		if err := screen.WritePNG(f, &m.PPU.Screen, pal, *scale); err != nil {
			f.Close()
			return err
		}
		if err := f.Close(); err != nil {
			return err
		}
	}
	if *until != "" && !found {
		return fmt.Errorf("the link port never said %q", *until)
	}
	return nil
}

// load reads a ROM and builds a machine for it.
func load(path string) (*gb.Machine, error) {
	rom, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return gb.New(rom)
}
