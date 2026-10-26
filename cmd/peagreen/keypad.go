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

// keypad turns key events into held buttons. With the kitty protocol the
// terminal reports releases, and buttons are held exactly as long as
// their keys.
type keypad struct {
	kitty bool
	down  gb.Button
	until [8]time.Time
}

// event takes a key event and says whether it was a button.
func (k *keypad) event(e term.Event, now time.Time) bool {
	b, ok := keyButtons[e.Key]
	if !ok {
		return false
	}
	if k.kitty {
		if e.Release {
			k.down &^= b
		} else {
			k.down |= b
		}
		return true
	}
	i := bit(b)
	hold := firstHold
	if now.Before(k.until[i]) {
		hold = repeatHold
	}
	k.until[i] = now.Add(hold)
	return true
}

// buttons is what is held at now.
func (k *keypad) buttons(now time.Time) gb.Button {
	if k.kitty {
		return k.down
	}
	var held gb.Button
	for i, t := range k.until {
		if now.Before(t) {
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
