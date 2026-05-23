//go:build !windows

package main

import (
	"os"
	"syscall"
	"unsafe"
)

func isRawModeSupported() bool { return true }

type termios struct {
	Iflag  uint32
	Oflag  uint32
	Cflag  uint32
	Lflag  uint32
	Cc     [20]uint8
	Ispeed uint32
	Ospeed uint32
}

var origTermios termios
var rawActive bool

func tcgetattr(fd uintptr, t *termios) error {
	_, _, errno := syscall.Syscall(syscall.SYS_IOCTL, fd, syscall.TCGETS, uintptr(unsafe.Pointer(t)))
	if errno != 0 { return errno }
	return nil
}

func tcsetattr(fd uintptr, t *termios) error {
	_, _, errno := syscall.Syscall(syscall.SYS_IOCTL, fd, syscall.TCSETS, uintptr(unsafe.Pointer(t)))
	if errno != 0 { return errno }
	return nil
}

const (
	ISIG = 0000001; ICANON = 0000002; ECHO = 0000010
	ECHOE = 0000020; ECHOK = 0000040; ECHONL = 0000100
	IEXTEN = 0100000; OPOST = 0000001; IXON = 0002000
	ICRNL = 0000400; BRKINT = 0000002; INPCK = 0000020
	ISTRIP = 0000040; CS8 = 0000060; VMIN = 6; VTIME = 5
)

func setRawMode() error {
	fd := os.Stdin.Fd()
	if err := tcgetattr(fd, &origTermios); err != nil { return err }
	rawActive = true
	raw := origTermios
	raw.Iflag &^= BRKINT | ICRNL | INPCK | ISTRIP | IXON
	raw.Oflag &^= OPOST
	raw.Cflag |= CS8
	raw.Lflag &^= ECHO | ECHOE | ECHOK | ECHONL | ICANON | IEXTEN | ISIG
	raw.Cc[VMIN] = 1
	raw.Cc[VTIME] = 0
	return tcsetattr(fd, &raw)
}

func restoreMode() {
	if rawActive {
		tcsetattr(os.Stdin.Fd(), &origTermios)
		rawActive = false
	}
}
