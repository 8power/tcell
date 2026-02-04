//go:build windows
// +build windows

package tcell

import (
	"fmt"
	"io"
)

var ErrNotSupported = fmt.Errorf("streaming screen not supported on this platform")

func NewStreamingScreen(rw io.ReadWriter, winSize func() (int, int)) (Screen, error) {
	// TBD: implement a Windows version if feasible.
	return nil, ErrNotSupported
}
