package main

import (
	"testing"

	"github.com/Rakshit-gen/peagreen/gb"
)

func TestRewind(t *testing.T) {
	r := newRewind(3)
	for i := range 5 {
		r.push(&gb.State{Version: i})
	}
	for _, want := range []int{4, 3, 2} {
		if s := r.pop(); s == nil || s.Version != want {
			t.Fatalf("popped %v, want state %d", s, want)
		}
	}
	if r.pop() != nil {
		t.Fatal("only the last three should be kept")
	}
	r.push(&gb.State{Version: 9})
	if s := r.pop(); s.Version != 9 {
		t.Fatal("pushing after emptying it")
	}
}
