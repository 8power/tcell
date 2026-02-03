//go:build aix || darwin || dragonfly || freebsd || linux || netbsd || openbsd || solaris || zos
// +build aix darwin dragonfly freebsd linux netbsd openbsd solaris zos

package tcell

import (
	"io"
	"sync"
)

type streamingTty struct {
	rw       io.ReadWriter     // e.g. SSH channel
	rwMutex  sync.Mutex        // mutex to protect access to rw
	winSize  func() (int, int) // width, height from remote side
	onResize func()            // optional callback
}

func (s *streamingTty) Fd() uintptr {
	// You can return 0 or a dummy; v2 mainly uses this for termios on real TTYs.
	return 0
}

func (s *streamingTty) Read(p []byte) (int, error) {
	return s.rw.Read(p)
}

func (s *streamingTty) Write(p []byte) (int, error) {
	s.rwMutex.Lock()
	defer s.rwMutex.Unlock()
	return s.rw.Write(p)
}

func (s *streamingTty) Close() error {
	if c, ok := s.rw.(io.Closer); ok {
		return c.Close()
	}
	return nil
}

// Window size & signals (you adapt to the exact v2 Tty API):

func (s *streamingTty) GetSize() (int, int, error) {
	w, h := s.winSize()
	return w, h, nil
}

func (s *streamingTty) NotifyResize(f func()) {
	if s.onResize != nil {
		s.onResize()
	}
}

func (s *streamingTty) Drain() error {
	// Read all available input

	return nil
}

func (s *streamingTty) WindowSize() (WindowSize, error) {
	w, h := s.winSize()
	return WindowSize{Width: w, Height: h}, nil
}

func (s *streamingTty) Start() error {
	return nil
}

func (s *streamingTty) Stop() error {
	return nil
}
