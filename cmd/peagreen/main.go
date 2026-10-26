// Command peagreen plays Game Boy games in a terminal.
package main

import (
	"fmt"
	"os"
	"strings"
)

// version is set by the release build.
var version = "dev"

const usage = `peagreen plays original Game Boy games in a terminal.

usage:
  peagreen play [flags] game.gb   play in the terminal (the default)
  peagreen run [flags] game.gb    run without a screen, for tests and screenshots
  peagreen info game.gb           show what the cartridge header says
  peagreen version

Run a command with -h for its flags.
`

func main() {
	args := os.Args[1:]
	if len(args) == 0 {
		fmt.Fprint(os.Stderr, usage)
		os.Exit(2)
	}
	var err error
	switch args[0] {
	case "play":
		err = play(args[1:])
	case "run":
		err = run(args[1:])
	case "info":
		err = info(args[1:])
	case "version", "-version", "--version":
		fmt.Println("peagreen", version)
	case "help", "-h", "-help", "--help":
		fmt.Print(usage)
	default:
		if strings.HasPrefix(args[0], "-") || strings.Contains(args[0], ".") {
			err = play(args)
		} else {
			err = fmt.Errorf("no command %q\n\n%s", args[0], usage)
		}
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, "peagreen:", err)
		os.Exit(1)
	}
}

func play(args []string) error { return fmt.Errorf("play isn't written yet") }
func run(args []string) error  { return fmt.Errorf("run isn't written yet") }
