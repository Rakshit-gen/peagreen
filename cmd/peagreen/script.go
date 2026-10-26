package main

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/Rakshit-gen/peagreen/gb"
)

var buttonNames = map[string]gb.Button{
	"right": gb.Right, "left": gb.Left, "up": gb.Up, "down": gb.Down,
	"a": gb.A, "b": gb.B, "select": gb.Select, "start": gb.Start,
}

// press is a button held from frame At for Frames frames.
type press struct {
	Button gb.Button
	At     int
	Frames int
}

// parseScript reads presses written like "start@60,a@120+10,left+down@200":
// buttons joined by +, the frame after @, and how many frames to hold
// after a second + (5 if left out).
func parseScript(s string) ([]press, error) {
	var out []press
	for _, item := range strings.Split(s, ",") {
		item = strings.TrimSpace(item)
		if item == "" {
			continue
		}
		names, when, ok := strings.Cut(item, "@")
		if !ok {
			return nil, fmt.Errorf("%q: want button@frame", item)
		}
		p := press{Frames: 5}
		for _, n := range strings.Split(names, "+") {
			b, ok := buttonNames[strings.ToLower(n)]
			if !ok {
				return nil, fmt.Errorf("%q: no button %q", item, n)
			}
			p.Button |= b
		}
		at, hold, hasHold := strings.Cut(when, "+")
		var err error
		if p.At, err = strconv.Atoi(at); err != nil || p.At < 0 {
			return nil, fmt.Errorf("%q: bad frame %q", item, at)
		}
		if hasHold {
			if p.Frames, err = strconv.Atoi(hold); err != nil || p.Frames < 1 {
				return nil, fmt.Errorf("%q: bad hold %q", item, hold)
			}
		}
		out = append(out, p)
	}
	return out, nil
}

// held is the buttons the script holds during a frame.
func held(script []press, frame int) gb.Button {
	var b gb.Button
	for _, p := range script {
		if frame >= p.At && frame < p.At+p.Frames {
			b |= p.Button
		}
	}
	return b
}
