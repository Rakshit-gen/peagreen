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

	"github.com/Rakshit-gen/peagreen/audio"
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

// Rewind keeps a state every rewindEvery frames for the last rewindSeconds,
// about 13 MB, and goes back one state each frame r is held, so it plays
// backwards at six times the speed.
const (
	rewindEvery   = 6
	rewindSeconds = 30
)

type player struct {
	m       *gb.Machine
	romPath string
	r       screen.Renderer
	out     *os.File
	keys    <-chan term.Event
	size    string // the -size flag, for refitting after a resize
	small   bool   // the terminal shrank below the frame; wait for it to grow
	pad     keypad
	back    key // r, held to rewind
	history *rewind
	bat     battery
	wav     *audio.WAV
	spk     *speaker
	quit    bool
	paused  bool
	help    bool // the key list is up, and the game waits
	fast    bool // run fastSpeed frames for each one shown
	note    string // a passing message for the status line
	noteEnd time.Time
	shown   string // the status line as last drawn
}

func play(args []string) error {
	fs := flag.NewFlagSet("play", flag.ContinueOnError)
	palName := fs.String("palette", "green", "palette: green, pocket, gray or amber")
	size := fs.String("size", "auto", "full (160x72 cells), half (80x36) or auto")
	wavPath := fs.String("wav", "", "record the sound to this WAV file")
	sound := fs.Bool("sound", true, "play sound through ffplay, sox, pacat or aplay")
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
	p := &player{m: m, romPath: fs.Arg(0), out: os.Stdout, size: *size, history: newRewind(rewindSeconds * 60 / rewindEvery)}
	p.bat = battery{path: savPath(fs.Arg(0)), last: append([]byte(nil), m.Cart.RAM...)}
	p.r.Palette = pal
	if err := p.fit(*size); err != nil {
		return err
	}
	if *sound {
		if p.spk, err = openSpeaker(); err != nil {
			p.say("%v", err)
		} else {
			m.EnableAudio(soundRate)
			defer p.spk.close()
		}
	}
	if *wavPath != "" {
		f, err := os.Create(*wavPath)
		if err != nil {
			return err
		}
		defer f.Close()
		m.EnableAudio(soundRate)
		if p.wav, err = audio.NewWAV(f, soundRate); err != nil {
			return err
		}
		defer func() {
			if err := p.wav.Close(); err != nil {
				fmt.Fprintln(os.Stderr, "peagreen: finishing the WAV:", err)
			}
		}()
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
	winch := make(chan os.Signal, 1)
	signal.Notify(winch, syscall.SIGWINCH)
	defer signal.Stop(winch)
	p.keys = term.Read(os.Stdin)
	err = p.loop(sigs, winch)
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

func (p *player) loop(sigs, winch <-chan os.Signal) error {
	next := time.Now()
	for frame := 0; !p.quit; frame++ {
		select {
		case <-sigs:
			return nil
		case <-winch:
			p.write("\x1b[0m\x1b[2J")
			p.shown = ""
			p.small, p.help = false, false
			if err := p.fit(p.size); err != nil {
				p.small = true
				fmt.Fprintf(p.out, "\x1b[1;1H%v", err)
			} else {
				// Redraw now: a paused game draws nothing by itself.
				p.out.Write(p.r.Frame(&p.m.PPU.Screen))
			}
		default:
		}
		p.input()
		switch {
		case p.small, p.help:
		case p.back.held(p.pad.kitty, time.Now()):
			p.stepBack()
		case !p.paused:
			p.m.SetButtons(p.pad.buttons(time.Now()))
			for range p.speed() {
				if p.m.PPU.Frames%rewindEvery == 0 {
					p.history.push(p.m.Save())
				}
				p.m.RunFrame()
				p.sound()
			}
			p.out.Write(p.r.Frame(&p.m.PPU.Screen))
		}
		if frame%saveEvery == 0 {
			if err := p.bat.save(p.m); err != nil {
				p.say("couldn't write the save: %v", err)
			}
		}
		if !p.small {
			p.status()
		}
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
			if e.Key == 'r' {
				p.back.event(e, p.pad.kitty, time.Now())
				continue
			}
			if e.Release || p.pad.event(e, time.Now()) {
				continue
			}
			switch e.Key {
			case term.KeyCtrlC, term.KeyEsc, 'q':
				p.quit = true
			case '?', 'h':
				p.toggleHelp()
			case 'p':
				p.paused = !p.paused
			case term.KeyTab:
				p.fast = !p.fast
			case 'c':
				p.nextPalette()
			case 'i':
				p.screenshot()
			case ',':
				p.saveState()
			case '.':
				p.loadState()
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

// sound passes on the samples of the frame just run.
func (p *player) sound() {
	samples := p.m.Audio()
	if p.spk != nil && !p.fast {
		p.spk.play(samples)
	}
	if p.wav != nil {
		if err := p.wav.Write(samples); err != nil {
			p.say("recording stopped: %v", err)
			p.wav = nil
		}
	}
}

// stepBack goes back to the newest state in the history.
func (p *player) stepBack() {
	s := p.history.pop()
	if s == nil {
		p.say("can't rewind further")
		return
	}
	if err := p.m.Load(s); err != nil {
		p.say("rewind: %v", err)
		return
	}
	p.out.Write(p.r.Frame(&p.m.PPU.Screen))
}

// statePath is where the save state lives, next to the ROM.
func (p *player) statePath() string {
	return strings.TrimSuffix(p.romPath, filepath.Ext(p.romPath)) + ".state"
}

func (p *player) saveState() {
	path := p.statePath()
	f, err := os.Create(path + ".tmp")
	if err == nil {
		err = gb.WriteState(f, p.m.Save())
		if cerr := f.Close(); err == nil {
			err = cerr
		}
	}
	if err == nil {
		err = os.Rename(path+".tmp", path)
	}
	if err != nil {
		p.say("couldn't save the state: %v", err)
		return
	}
	p.say("state saved")
}

func (p *player) loadState() {
	f, err := os.Open(p.statePath())
	if errors.Is(err, fs.ErrNotExist) {
		p.say("no saved state yet; , saves one")
		return
	}
	var s *gb.State
	if err == nil {
		s, err = gb.ReadState(f)
		f.Close()
	}
	if err == nil {
		err = p.m.Load(s)
	}
	if err != nil {
		p.say("couldn't load the state: %v", err)
		return
	}
	p.out.Write(p.r.Frame(&p.m.PPU.Screen))
	p.say("state loaded")
}

var helpText = []string{
	"arrows or wasd   d-pad",
	"x or k           A",
	"z or j           B",
	"enter            Start",
	"backspace        Select",
	"",
	"p                pause",
	"tab              fast-forward",
	"r (hold)         rewind",
	", and .          save and load state",
	"c                next palette",
	"i                screenshot",
	"q or esc         quit",
	"",
	"? or h           close this",
}

// toggleHelp shows the keys over the frame, or takes them away.
func (p *player) toggleHelp() {
	p.help = !p.help
	if !p.help {
		p.r.Invalidate()
		p.out.Write(p.r.Frame(&p.m.PPU.Screen))
		return
	}
	cols, rows := p.r.Size()
	width := 0
	for _, l := range helpText {
		width = max(width, len(l))
	}
	width += 4
	top := p.r.Row + (rows-len(helpText)-2)/2
	left := p.r.Col + (cols-width)/2
	p.write("\x1b[0m")
	for i := -1; i <= len(helpText); i++ {
		l := ""
		if i >= 0 && i < len(helpText) {
			l = helpText[i]
		}
		fmt.Fprintf(p.out, "\x1b[%d;%dH  %-*s  ", top+1+i, left, width-4, l)
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
		line += "  ? keys  p pause  tab fast  q quit"
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
