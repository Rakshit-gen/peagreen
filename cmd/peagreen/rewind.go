package main

import "github.com/Rakshit-gen/peagreen/gb"

// rewind keeps the last states taken, oldest dropped first.
type rewind struct {
	states []*gb.State
	head   int // where the next push goes
	n      int
}

func newRewind(size int) *rewind { return &rewind{states: make([]*gb.State, size)} }

func (r *rewind) push(s *gb.State) {
	r.states[r.head] = s
	r.head = (r.head + 1) % len(r.states)
	r.n = min(r.n+1, len(r.states))
}

// pop takes back the newest state, or nil when none is left.
func (r *rewind) pop() *gb.State {
	if r.n == 0 {
		return nil
	}
	r.head = (r.head - 1 + len(r.states)) % len(r.states)
	r.n--
	s := r.states[r.head]
	r.states[r.head] = nil
	return s
}
