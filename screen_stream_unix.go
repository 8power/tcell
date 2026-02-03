//go:build aix || darwin || dragonfly || freebsd || linux || netbsd || openbsd || solaris || zos
// +build aix darwin dragonfly freebsd linux netbsd openbsd solaris zos

package tcell

import (
	"io"
)

func NewStreamingScreen(rw io.ReadWriter, winSize func() (int, int)) (Screen, error) {
	tty := &streamingTty{
		rw:      rw,
		winSize: winSize,
	}
	return NewTerminfoScreenFromTty(tty)
}
