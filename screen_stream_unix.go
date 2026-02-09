//go:build aix || darwin || dragonfly || freebsd || linux || netbsd || openbsd || solaris || zos
// +build aix darwin dragonfly freebsd linux netbsd openbsd solaris zos

package tcell

import (
	"net"
)

func NewStreamingScreen(conn net.Conn) (Screen, error) {
	tty, _ := NewStreamingTty(conn)
	return NewTerminfoScreenFromTty(tty)
}
