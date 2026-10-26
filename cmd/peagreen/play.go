package main

import (
	"errors"
	"flag"
	"fmt"
	"io/fs"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/Rakshit-gen/peagreen/gb"
	"github.com/Rakshit-gen/peagreen/screen"
	"github.com/Rakshit-gen/peagreen/term"
)

// framePeriod is how long a frame lasts on the hardware: 70224 cycles of
// a 4.194304 MHz clock, about 59.73 frames a second.
const framePeriod = time.Second * 70224 / 4194304

const fastSpeed = 4

// saveEvery is how often, in frames, play checks whether the game wrote to
// its battery RAM and writes the save file if it did.
const saveEvery = 120

type player struct {
	m       *gb.Machine
	romPath string
	r       screen.Renderer
	out     *os.File
	keys    <-chan term.Event
	pad     keypad
	bat     battery
	quit    bool
	paused  bool
	fast    bool // run fastSpeed frames for each one shown
	note    string // a passing message for the status line
	noteEnd time.Time
	shown   string // the status line as last drawn
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
	if err := loadBattery(m, fs.Arg(0)); err != nil {
		return err
	}
	p := &player{m: m, romPath: fs.Arg(0), out: os.Stdout}
	p.bat = battery{path: savPath(fs.Arg(0)), last: append([]byte(nil), m.Cart.RAM...)}
	p.r.Palette = pal
	if err := p.fit(*size); err != nil {
		return err
	}
	restore, err := term.MakeRaw(in)
	if err != nil {
		return err
	}
	// Alternate screen, cursor hidden, and a question for the terminal:
	// does it speak the kitty keyboard protocol? All undone on the way out.
	p.write("\x1b[?1049h\x1b[?25l\x1b[2J\x1b[?u")
	defer func() {
		if p.pad.kitty {
			p.write("\x1b[<u")
		}
		p.write("\x1b[0m\x1b[2J\x1b[?25h\x1b[?1049l")
		restore()
	}()
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGTERM, syscall.SIGHUP, syscall.SIGINT)
	defer signal.Stop(sigs)
	p.keys = term.Read(os.Stdin)
	err = p.loop(sigs)
	// Always write the save on the way out, so a clock's time is kept.
	p.bat.last = nil
	if serr := p.bat.save(m); err == nil {
		err = serr
	}
	return err
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
	for frame := 0; !p.quit; frame++ {
		select {
		case <-sigs:
			return nil
		default:
		}
		p.input()
		if !p.paused {
			p.m.SetButtons(p.pad.buttons(time.Now()))
			for range p.speed() {
				p.m.RunFrame()
			}
			p.out.Write(p.r.Frame(&p.m.PPU.Screen))
		}
		if frame%saveEvery == 0 {
			if err := p.bat.save(p.m); err != nil {
				p.say("couldn't write the save: %v", err)
			}
		}
		p.status()
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
			if e.Key == term.KeyKitty && !p.pad.kitty {
				// It does. Ask for key releases, and for every key as an
				// escape code so releases come for letters too.
				p.pad.kitty = true
				p.write("\x1b[>11u")
				continue
			}
			if e.Release || p.pad.event(e, time.Now()) {
				continue
			}
			switch e.Key {
			case term.KeyCtrlC, term.KeyEsc, 'q':
				p.quit = true
			case 'p':
				p.paused = !p.paused
			case term.KeyTab:
				p.fast = !p.fast
			case 'c':
				p.nextPalette()
			case 'i':
				p.screenshot()
			}
		default:
			return
		}
	}
}

func (p *player) speed() int {
	if p.fast {
		return fastSpeed
	}
	return 1
}

// nextPalette switches to the next built-in palette and redraws.
func (p *player) nextPalette() {
	i := 0
	for j, pal := range screen.Palettes {
		if pal.Name == p.r.Palette.Name {
			i = j + 1
		}
	}
	p.r.Palette = screen.Palettes[i%len(screen.Palettes)]
	p.r.Invalidate()
	p.out.Write(p.r.Frame(&p.m.PPU.Screen))
	p.say("palette %s", p.r.Palette.Name)
}

// screenshot saves the screen as game-1.png, game-2.png and so on, next
// to the ROM.
func (p *player) screenshot() {
	base := strings.TrimSuffix(p.romPath, filepath.Ext(p.romPath))
	for n := 1; ; n++ {
		path := fmt.Sprintf("%s-%d.png", base, n)
		f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644)
		if errors.Is(err, fs.ErrExist) {
			continue
		}
		if err == nil {
			err = screen.WritePNG(f, &p.m.PPU.Screen, p.r.Palette, 3)
			if cerr := f.Close(); err == nil {
				err = cerr
			}
		}
		if err != nil {
			p.say("screenshot failed: %v", err)
			return
		}
		p.say("saved %s", filepath.Base(path))
		return
	}
}

// say shows a message on the status line for a few seconds.
func (p *player) say(format string, args ...any) {
	p.note = fmt.Sprintf(format, args...)
	p.noteEnd = time.Now().Add(3 * time.Second)
}

// status draws the line under the frame, when it changed.
func (p *player) status() {
	line := p.m.Cart.Header.Title
	if p.paused {
		line += "  paused"
	} else if p.fast {
		line += fmt.Sprintf("  x%d", fastSpeed)
	}
	if p.note != "" && time.Now().Before(p.noteEnd) {
		line += "  " + p.note
	} else {
		line += "  p pause  tab fast  q quit"
	}
	cols, rows := p.r.Size()
	if len(line) > cols {
		line = line[:cols]
	}
	if line == p.shown {
		return
	}
	p.shown = line
	fmt.Fprintf(p.out, "\x1b[%d;%dH\x1b[0m\x1b[2K%s", p.r.Row+rows, p.r.Col, line)
}
