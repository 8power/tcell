//go:build windows
// +build windows

package tcell

import (
	"fmt"
	"net"
)

var ErrNotSupported = fmt.Errorf("streaming screen not supported on this platform")

func NewStreamingScreen(conn net.Conn) (Screen, error) {
	// TBD: implement a Windows version if feasible.
	return nil, ErrNotSupported
}
