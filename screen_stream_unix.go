//go:build aix || darwin || dragonfly || freebsd || linux || netbsd || openbsd || solaris || zos
// +build aix darwin dragonfly freebsd linux netbsd openbsd solaris zos

package tcell

import (
	"io"
)

func NewStreamingScreen(in <-chan []byte, out chan<- []byte, closer io.Closer) (Screen, error) {
	tty := NewStreamingTty(in, out, closer)
	return NewTerminfoScreenFromTty(tty)
}
