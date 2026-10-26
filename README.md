# peagreen

A Game Boy emulator that plays in your terminal.

peagreen emulates the original Game Boy (the DMG, the grey brick with the
green screen) and draws it with half-block characters, two pixels to a
character cell. It has sound, save files, save states, rewind and
fast-forward, and it's written in Go with nothing outside the standard
library.

```
peagreen tetris.gb
```

## Install

With the install script, which puts the binary in `~/.local/bin`:

```
curl -fsSL https://raw.githubusercontent.com/Rakshit-gen/peagreen/main/scripts/install.sh | sh
```

With Go 1.25 or newer:

```
go install github.com/Rakshit-gen/peagreen/cmd/peagreen@latest
```

Or from a clone, with `make install`.

Releases are built for Linux and macOS, on amd64 and arm64.

## Playing

| Key | Game Boy |
| --- | --- |
| arrows or WASD | d-pad |
| x or k | A |
| z or j | B |
| Enter | Start |
| Backspace | Select |

| Key | peagreen |
| --- | --- |
| p | pause |
| Tab | fast-forward (4x) |
| r, held | rewind, up to 30 seconds |
| `,` and `.` | save and load a state |
| 1 to 9 | pick the state slot |
| m | sound off and on |
| g | ghosting off and on |
| c | next palette |
| i | screenshot |
| ? or h | show the keys |
| q or Esc | quit |

Terminals only send key presses, not releases, so peagreen counts a key
as held for a moment after each press, and longer after the first, to
cover the gap before auto-repeat. Terminals that speak the kitty keyboard
protocol (kitty, WezTerm, foot, Ghostty and others) send releases too, and
peagreen uses them when it finds them, which makes the controls much
tighter.

The frame takes 160 columns by 72 rows. In a smaller terminal peagreen
draws it at half size, 80 by 36, and it refits when the window changes.

### Files next to the ROM

- `game.sav` holds the cartridge's battery RAM, for games that save. It's
  written every two seconds or so when the game changed it, and on quit.
- `game.state` is the save state in slot 1, `game-2.state` slot 2 and so
  on. A later peagreen may refuse a state if the format changed; the
  `.sav` file always carries over.
- `game-1.png`, `game-2.png` and so on are screenshots, at 3x.

### Flags

```
peagreen play [flags] game.gb

  -palette  green, pocket, gray or amber (default green)
  -size     full, half or auto (default auto)
  -colors   24bit, 256, or auto to go by $COLORTERM (default auto)
  -sound    play sound (default true)
  -ghost    blend each frame with the one before, like the DMG's slow screen
  -wav      record the sound to a WAV file
```

`-ghost` matters for games that flicker sprites on alternate frames to
get past the limit of ten per line. On the real screen, which is slow to
change, those sprites look solid; without it they blink.

## Sound

Go's standard library has no way to reach a sound card, so peagreen pipes
its sound into the first of these it finds: `ffplay` (from ffmpeg), sox's
`play`, `pacat` (PulseAudio and PipeWire) or `aplay` (ALSA). On macOS,
`brew install ffmpeg` or `brew install sox` gives you one. Without any of
them the game plays silently and says why on the status line.

If the player falls behind, peagreen drops sound rather than slowing the
game down. Fast-forward is silent.

## Scripting

`peagreen run` runs a game without a screen, for tests, screenshots and
recordings:

```
peagreen run -frames 300 -press "start@60,a@120+10" -png title.png game.gb
peagreen run -frames 5000 -serial -until Passed cpu_instrs.gb
```

`-press` holds buttons from a frame for a number of frames (5 if left
out), and `-until` stops as soon as the link port output contains the
text, failing if it never does. `-wav` records the sound.

`peagreen info game.gb` prints what the cartridge header says.
