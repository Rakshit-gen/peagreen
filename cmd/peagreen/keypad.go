package main

import (
	"time"

	"github.com/Rakshit-gen/peagreen/gb"
	"github.com/Rakshit-gen/peagreen/term"
)

var keyButtons = map[term.Key]gb.Button{
	term.KeyUp: gb.Up, term.KeyDown: gb.Down, term.KeyLeft: gb.Left, term.KeyRight: gb.Right,
	'w': gb.Up, 's': gb.Down, 'a': gb.Left, 'd': gb.Right,
	'x': gb.A, 'k': gb.A, 'z': gb.B, 'j': gb.B,
	term.KeyEnter: gb.Start, term.KeyBackspace: gb.Select,
}

// Most terminals send a key once when it goes down and then, after a
// pause, again and again while it stays down, but never say when it comes
// up. So a key counts as held for firstHold after it goes down, long
// enough to reach the first repeat, and for repeatHold after each repeat.
const (
	firstHold  = 450 * time.Millisecond
	repeatHold = 100 * time.Millisecond
)

// key tracks whether one key is held. With the kitty protocol the
// terminal reports releases, and a key is held exactly until it comes up.
type key struct {
	down  bool
	until time.Time
}

func (h *key) event(e term.Event, kitty bool, now time.Time) {
	if kitty {
		h.down = !e.Release
		return
	}
	hold := firstHold
	if now.Before(h.until) {
		hold = repeatHold
	}
	h.until = now.Add(hold)
}

func (h *key) held(kitty bool, now time.Time) bool {
	if kitty {
		return h.down
	}
	return now.Before(h.until)
}

// keypad turns key events into held buttons.
type keypad struct {
	kitty bool
	keys  [8]key // one for each button, by bit
}

// event takes a key event and says whether it was a button.
func (k *keypad) event(e term.Event, now time.Time) bool {
	b, ok := keyButtons[e.Key]
	if !ok {
		return false
	}
	if o := opposite[b]; o != 0 && !e.Release {
		k.keys[bit(o)] = key{}
	}
	k.keys[bit(b)].event(e, k.kitty, now)
	return true
}

// A real d-pad can't press left and right, or up and down, at once, and
// some games misbehave if they read both. The newer press wins.
var opposite = map[gb.Button]gb.Button{gb.Left: gb.Right, gb.Right: gb.Left, gb.Up: gb.Down, gb.Down: gb.Up}

// buttons is what is held at now.
func (k *keypad) buttons(now time.Time) gb.Button {
	var held gb.Button
	for i := range k.keys {
		if k.keys[i].held(k.kitty, now) {
			held |= 1 << i
		}
	}
	return held
}

func bit(b gb.Button) int {
	i := 0
	for b > 1 {
		b >>= 1
		i++
	}
	return i
}
