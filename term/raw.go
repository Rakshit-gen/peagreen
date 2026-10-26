//go:build darwin || linux

// Package term drives the terminal: raw mode, its size, and reading keys.
package term

import (
	"syscall"
	"unsafe"
)

func ioctl(fd int, req uintptr, arg unsafe.Pointer) error {
	if _, _, e := syscall.Syscall(syscall.SYS_IOCTL, uintptr(fd), req, uintptr(arg)); e != 0 {
		return e
	}
	return nil
}

// MakeRaw puts the terminal on fd into raw mode: no echo, no line
// buffering, no signals from ctrl-c. It returns a function that puts back
// the mode it found.
func MakeRaw(fd int) (restore func() error, err error) {
	var old syscall.Termios
	if err := ioctl(fd, ioctlGet, unsafe.Pointer(&old)); err != nil {
		return nil, err
	}
	t := old
	t.Iflag &^= syscall.IGNBRK | syscall.BRKINT | syscall.PARMRK | syscall.ISTRIP |
		syscall.INLCR | syscall.IGNCR | syscall.ICRNL | syscall.IXON
	t.Lflag &^= syscall.ECHO | syscall.ECHONL | syscall.ICANON | syscall.ISIG | syscall.IEXTEN
	t.Cflag &^= syscall.CSIZE | syscall.PARENB
	t.Cflag |= syscall.CS8
	t.Cc[syscall.VMIN] = 1
	t.Cc[syscall.VTIME] = 0
	if err := ioctl(fd, ioctlSet, unsafe.Pointer(&t)); err != nil {
		return nil, err
	}
	return func() error { return ioctl(fd, ioctlSet, unsafe.Pointer(&old)) }, nil
}

// Size returns the terminal's columns and rows.
func Size(fd int) (cols, rows int, err error) {
	var ws struct{ Row, Col, X, Y uint16 }
	if err := ioctl(fd, syscall.TIOCGWINSZ, unsafe.Pointer(&ws)); err != nil {
		return 0, 0, err
	}
	return int(ws.Col), int(ws.Row), nil
}

// IsTerminal reports whether fd is a terminal.
func IsTerminal(fd int) bool {
	var t syscall.Termios
	return ioctl(fd, ioctlGet, unsafe.Pointer(&t)) == nil
}
