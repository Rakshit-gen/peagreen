package main

import (
	"testing"
	"time"

	"github.com/Rakshit-gen/peagreen/gb"
	"github.com/Rakshit-gen/peagreen/term"
)

func TestKeypadRepeats(t *testing.T) {
	var k keypad
	t0 := time.Unix(0, 0)
	at := func(ms int) time.Time { return t0.Add(time.Duration(ms) * time.Millisecond) }
	if !k.event(term.Event{Key: 'x'}, t0) || k.event(term.Event{Key: 'm'}, t0) {
		t.Fatal("x is A, m is nothing")
	}
	if k.buttons(at(400)) != gb.A {
		t.Fatal("A should be held until the first repeat could come")
	}
	if k.buttons(at(500)) != 0 {
		t.Fatal("A should let go when no repeat came")
	}
	k.event(term.Event{Key: term.KeyLeft}, at(1000))
	k.event(term.Event{Key: term.KeyLeft}, at(1300)) // the terminal's repeat
	if k.buttons(at(1420)) != 0 {
		t.Fatal("a repeat should hold for only a short while")
	}
	if k.buttons(at(1350)) != gb.Left {
		t.Fatal("Left should be held just after a repeat")
	}
}

func TestKeypadKitty(t *testing.T) {
	k := keypad{kitty: true}
	now := time.Unix(0, 0)
	k.event(term.Event{Key: term.KeyEnter}, now)
	k.event(term.Event{Key: 'w'}, now)
	if k.buttons(now.Add(time.Hour)) != gb.Start|gb.Up {
		t.Fatal("with kitty, keys stay down until released")
	}
	k.event(term.Event{Key: 'w', Release: true}, now)
	if k.buttons(now) != gb.Start {
		t.Fatal("releasing w should let go of Up only")
	}
}
