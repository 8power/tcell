//go:build aix || darwin || dragonfly || freebsd || linux || netbsd || openbsd || solaris || zos
// +build aix darwin dragonfly freebsd linux netbsd openbsd solaris zos

package tcell

import (
	"net"
)

func NewStreamingScreen(in chan []byte, out chan []byte, conn net.Conn) (Screen, error) {
	tty, _ := NewStreamingTty(in, out, conn)
	return NewTerminfoScreenFromTty(tty)
}
