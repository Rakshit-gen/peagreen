package gb

import (
	"encoding/gob"
	"errors"
	"fmt"
	"io"
)

// State is a snapshot of a running machine, for save states and rewind.
// It holds everything but the ROM.
type State struct {
	Version int
	M       Machine // with Cart nil; the cartridge is below
	Cart    Cart
}

// stateVersion goes up when State changes shape, so old save states are
// refused instead of loaded wrong.
const stateVersion = 1

// Save takes a snapshot.
func (m *Machine) Save() *State {
	s := &State{Version: stateVersion, M: *m, Cart: *m.Cart}
	s.M.Cart, s.M.serialOut = nil, nil
	s.M.APU.samples = nil
	s.Cart.rom = nil
	s.Cart.RAM = append([]byte(nil), m.Cart.RAM...)
	return s
}

// Load puts a snapshot back. It must come from the same game.
func (m *Machine) Load(s *State) error {
	if s.Version != stateVersion {
		return fmt.Errorf("the state is version %d, this build reads version %d", s.Version, stateVersion)
	}
	c := m.Cart
	if s.Cart.Header != c.Header || len(s.Cart.RAM) != len(c.RAM) {
		return errors.New("the state is from a different game")
	}
	// Keep what isn't the game's: the ROM, the mapper, where the link
	// port and sound go.
	live, out := m.APU, m.serialOut
	*m = s.M
	m.Cart, m.serialOut = c, out
	a := &m.APU
	a.rate, a.frac, a.sumL, a.sumR, a.n = live.rate, live.frac, live.sumL, live.sumR, live.n
	a.capL, a.capR, a.highPass, a.samples = live.capL, live.capR, live.highPass, live.samples
	ram := c.RAM
	cart := s.Cart
	cart.rom, cart.banks, cart.kind, cart.multi = c.rom, c.banks, c.kind, c.multi
	*c = cart
	c.RAM = ram
	copy(c.RAM, s.Cart.RAM)
	return nil
}

// WriteState writes a snapshot to w.
func WriteState(w io.Writer, s *State) error { return gob.NewEncoder(w).Encode(s) }

// ReadState reads a snapshot written by WriteState.
func ReadState(r io.Reader) (*State, error) {
	var s State
	if err := gob.NewDecoder(r).Decode(&s); err != nil {
		return nil, fmt.Errorf("reading the state: %w", err)
	}
	return &s, nil
}
