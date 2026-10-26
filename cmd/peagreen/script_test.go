package main

import (
	"testing"

	"github.com/Rakshit-gen/peagreen/gb"
)

func TestParseScript(t *testing.T) {
	s, err := parseScript("start@60, a@120+10,left+down@200")
	if err != nil {
		t.Fatal(err)
	}
	if len(s) != 3 || s[1] != (press{gb.A, 120, 10}) || s[2].Button != gb.Left|gb.Down {
		t.Fatalf("got %+v", s)
	}
	for frame, want := range map[int]gb.Button{59: 0, 60: gb.Start, 64: gb.Start, 65: 0, 129: gb.A, 130: 0, 202: gb.Left | gb.Down} {
		if got := held(s, frame); got != want {
			t.Errorf("frame %d: %v, want %v", frame, got, want)
		}
	}
	for _, bad := range []string{"start", "jump@3", "a@x", "a@3+0"} {
		if _, err := parseScript(bad); err == nil {
			t.Errorf("%q should fail", bad)
		}
	}
}
