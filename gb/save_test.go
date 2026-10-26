package gb

import (
	"testing"
	"time"
)

func TestSaveRoundTrip(t *testing.T) {
	c := bankedCart(t, 0x10, 2, 3) // MBC3 with clock, RAM and battery
	c.RAM[5] = 0x42
	c.RTC.H, c.RTC.M, c.RTC.S, c.RTC.Days = 10, 20, 30, 300
	then := time.Unix(1_700_000_000, 0)
	b := c.SaveData(then)
	if len(b) != 32<<10+48 {
		t.Fatalf("save is %d bytes, want RAM plus 48", len(b))
	}
	d := bankedCart(t, 0x10, 2, 3)
	// Load it 1 day, 1 hour and 5 seconds later.
	if err := d.LoadSave(b, then.Add(25*time.Hour+5*time.Second)); err != nil {
		t.Fatal(err)
	}
	r := d.RTC
	if d.RAM[5] != 0x42 || r.Days != 301 || r.H != 11 || r.M != 20 || r.S != 35 {
		t.Fatalf("day %d %d:%d:%d, ram %x", r.Days, r.H, r.M, r.S, d.RAM[5])
	}
}

func TestSaveNoBattery(t *testing.T) {
	c := bankedCart(t, 0x02, 2, 2) // MBC1+RAM, no battery
	if c.SaveData(time.Now()) != nil {
		t.Fatal("no battery, nothing to save")
	}
	if err := c.LoadSave(make([]byte, 10), time.Now()); err == nil {
		t.Fatal("a save shorter than RAM should fail")
	}
}
