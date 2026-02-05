//go:build aix || darwin || dragonfly || freebsd || linux || netbsd || openbsd || solaris || zos
// +build aix darwin dragonfly freebsd linux netbsd openbsd solaris zos

package tcell

import (
	"io"
)

func NewStreamingScreen(rw io.ReadWriter) (Screen, error) {
	tty := NewStreamingTty(rw)
	return NewTerminfoScreenFromTty(tty)
}
