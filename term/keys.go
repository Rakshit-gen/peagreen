package term

// Key is a key: a rune for printable keys, or one of the named ones below.
type Key rune

// Named keys, below any rune.
const (
	KeyUp Key = -1 - iota
	KeyDown
	KeyLeft
	KeyRight
	KeyEnter
	KeyBackspace
	KeyTab
	KeyEsc
	KeyCtrlC
	KeyF1
	KeyF2
	KeyF3
	KeyF4
	KeyF5
	KeyF6
	KeyF7
	KeyF8
	KeyF9
	KeyF10
	KeyF11
	KeyF12
	KeyUnknown

	// KeyKitty isn't a key: it's the terminal answering that it speaks
	// the kitty keyboard protocol.
	KeyKitty
)

// Event is a key going down, repeating or coming up. Most terminals only
// say a key went down (and again while it auto-repeats); terminals with
// the kitty keyboard protocol also say when it comes up.
type Event struct {
	Key     Key
	Repeat  bool
	Release bool
}

// Parse splits input into events. A sequence cut off at the end of buf is
// returned in rest, to be parsed again with whatever comes next.
func Parse(buf []byte) (events []Event, rest []byte) {
	for len(buf) > 0 {
		ev, n := parseOne(buf)
		if n == 0 {
			return events, buf
		}
		if ev.Key != 0 {
			events = append(events, ev)
		}
		buf = buf[n:]
	}
	return events, nil
}

// parseOne reads one event from the start of buf and says how many bytes
// it used, 0 if buf ends partway through a sequence. A zero Key means the
// bytes meant nothing worth reporting.
func parseOne(buf []byte) (Event, int) {
	b := buf[0]
	switch {
	case b == 0x1B:
		return parseEsc(buf)
	case b == '\r' || b == '\n':
		return Event{Key: KeyEnter}, 1
	case b == 0x7F || b == 0x08:
		return Event{Key: KeyBackspace}, 1
	case b == '\t':
		return Event{Key: KeyTab}, 1
	case b == 0x03:
		return Event{Key: KeyCtrlC}, 1
	case b < 0x20:
		return Event{}, 1
	case b < 0x80:
		return Event{Key: Key(b)}, 1
	}
	// A UTF-8 rune: skip its bytes; nothing here needs non-ASCII keys.
	n := 1
	for n < len(buf) && n < 4 && buf[n]&0xC0 == 0x80 {
		n++
	}
	return Event{Key: KeyUnknown}, n
}

// parseEsc reads a sequence starting with ESC. A lone ESC at the end of
// the input is ambiguous (the Esc key, or the start of a sequence still on
// its way); the reader waits a moment before calling it the Esc key.
func parseEsc(buf []byte) (Event, int) {
	if len(buf) < 2 {
		return Event{}, 0
	}
	switch buf[1] {
	case '[':
		return parseCSI(buf)
	case 'O': // SS3: arrows in application mode, F1-F4
		if len(buf) < 3 {
			return Event{}, 0
		}
		return Event{Key: finalKey(buf[2], 1)}, 3
	}
	return Event{Key: KeyEsc}, 1
}

// parseCSI reads ESC [ params final.
func parseCSI(buf []byte) (Event, int) {
	i := 2
	for i < len(buf) && (buf[i] >= 0x30 && buf[i] <= 0x3F) {
		i++
	}
	if i == len(buf) {
		return Event{}, 0
	}
	params, final := string(buf[2:i]), buf[i]
	n := i + 1
	nums := splitParams(params)
	first := 1
	if len(nums) > 0 && len(nums[0]) > 0 {
		first = nums[0][0]
	}
	if final == '~' {
		return Event{Key: tildeKey(first)}, n
	}
	return Event{Key: finalKey(final, first)}, n
}

// splitParams turns "1;5:3" into [[1] [5 3]].
func splitParams(s string) [][]int {
	var out [][]int
	var cur []int
	num, have := 0, false
	flush := func() {
		if have {
			cur = append(cur, num)
		}
		num, have = 0, false
	}
	for i := 0; i < len(s); i++ {
		switch c := s[i]; {
		case c >= '0' && c <= '9':
			num, have = num*10+int(c-'0'), true
		case c == ':':
			flush()
		case c == ';':
			flush()
			out = append(out, cur)
			cur = nil
		}
	}
	flush()
	return append(out, cur)
}

func finalKey(final byte, _ int) Key {
	switch final {
	case 'A':
		return KeyUp
	case 'B':
		return KeyDown
	case 'C':
		return KeyRight
	case 'D':
		return KeyLeft
	case 'P':
		return KeyF1
	case 'Q':
		return KeyF2
	case 'R':
		return KeyF3
	case 'S':
		return KeyF4
	}
	return KeyUnknown
}

func tildeKey(n int) Key {
	switch n {
	case 11:
		return KeyF1
	case 12:
		return KeyF2
	case 13:
		return KeyF3
	case 14:
		return KeyF4
	case 15:
		return KeyF5
	case 17:
		return KeyF6
	case 18:
		return KeyF7
	case 19:
		return KeyF8
	case 20:
		return KeyF9
	case 21:
		return KeyF10
	case 23:
		return KeyF11
	case 24:
		return KeyF12
	}
	return KeyUnknown
}
