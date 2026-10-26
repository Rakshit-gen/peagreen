package main

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/Rakshit-gen/peagreen/gb"
	"github.com/Rakshit-gen/peagreen/screen"
	"github.com/Rakshit-gen/peagreen/term"
)

// framePeriod is how long a frame lasts on the hardware: 70224 cycles of
// a 4.194304 MHz clock, about 59.73 frames a second.
const framePeriod = time.Second * 70224 / 4194304

type player struct {
	m       *gb.Machine
	romPath string
	r       screen.Renderer
	out     *os.File
	keys    <-chan term.Event
	quit    bool
}

func play(args []string) error {
	fs := flag.NewFlagSet("play", flag.ContinueOnError)
	palName := fs.String("palette", "green", "palette: green, pocket, gray or amber")
	size := fs.String("size", "auto", "full (160x72 cells), half (80x36) or auto")
	fs.Usage = func() {
		fmt.Fprintln(fs.Output(), "usage: peagreen play [flags] game.gb")
		fs.PrintDefaults()
	}
	if err := fs.Parse(args); err != nil {
		return err
	}
	if fs.NArg() != 1 {
		fs.Usage()
		return errors.New("play takes one ROM")
	}
	pal, err := screen.Find(*palName)
	if err != nil {
		return err
	}
	in, out := int(os.Stdin.Fd()), int(os.Stdout.Fd())
	if !term.IsTerminal(in) || !term.IsTerminal(out) {
		return errors.New("play needs a terminal; use peagreen run for scripts")
	}
	m, err := load(fs.Arg(0))
	if err != nil {
		return err
	}
	p := &player{m: m, romPath: fs.Arg(0), out: os.Stdout}
	p.r.Palette = pal
	if err := p.fit(*size); err != nil {
		return err
	}
	restore, err := term.MakeRaw(in)
	if err != nil {
		return err
	}
	// Alternate screen, cursor hidden; all undone on the way out.
	p.write("\x1b[?1049h\x1b[?25l\x1b[2J")
	defer func() {
		p.write("\x1b[0m\x1b[2J\x1b[?25h\x1b[?1049l")
		restore()
	}()
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGTERM, syscall.SIGHUP, syscall.SIGINT)
	defer signal.Stop(sigs)
	p.keys = term.Read(os.Stdin)
	return p.loop(sigs)
}

// fit picks the frame size for the terminal and centres the frame.
func (p *player) fit(size string) error {
	cols, rows, err := term.Size(int(p.out.Fd()))
	if err != nil {
		return err
	}
	switch size {
	case "full":
		p.r.Half = false
	case "half":
		p.r.Half = true
	case "auto":
		p.r.Half = cols < gb.Width || rows < gb.Height/2+1
	default:
		return fmt.Errorf("-size is full, half or auto, not %q", size)
	}
	fc, fr := p.r.Size()
	if cols < fc || rows < fr+1 {
		return fmt.Errorf("the terminal is %dx%d; peagreen needs %dx%d", cols, rows, fc, fr+1)
	}
	p.r.Col = (cols-fc)/2 + 1
	p.r.Row = (rows-fr-1)/2 + 1
	p.r.Invalidate()
	return nil
}

func (p *player) write(s string) { p.out.WriteString(s) }

func (p *player) loop(sigs <-chan os.Signal) error {
	next := time.Now()
	for !p.quit {
		select {
		case <-sigs:
			return nil
		default:
		}
		p.input()
		p.m.RunFrame()
		p.out.Write(p.r.Frame(&p.m.PPU.Screen))
		next = next.Add(framePeriod)
		if d := time.Until(next); d > 0 {
			time.Sleep(d)
		} else if d < -10*framePeriod {
			next = time.Now() // fell far behind; don't race to catch up
		}
	}
	return nil
}

// input handles the key events that arrived since the last frame.
func (p *player) input() {
	for {
		select {
		case e, ok := <-p.keys:
			if !ok {
				p.quit = true
				return
			}
			switch e.Key {
			case term.KeyCtrlC, term.KeyEsc, 'q':
				p.quit = true
			}
		default:
			return
		}
	}
}
