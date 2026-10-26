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
