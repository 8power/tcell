//go:build aix || darwin || dragonfly || freebsd || linux || netbsd || openbsd || solaris || zos
// +build aix darwin dragonfly freebsd linux netbsd openbsd solaris zos

package tcell

import (
	"io"
	"net"
	"sync"
)

type streamingTty struct {
	io.Closer // e.g. SSH channel
	inPipe    chan []byte
	outPipe   chan []byte
	rwMutex   sync.Mutex // mutex to protect access to rw
	width     int        // character width handling
	height    int        // character height handling
	onResize  func()     // callback for resize events
}

func NewStreamingTty(conn net.Conn) (Tty, *TelnetIO) {
	s := &streamingTty{
		Closer:  conn,
		inPipe:  make(chan []byte, 1024),
		outPipe: make(chan []byte, 1024),
		width:   80,
		height:  24,
	}

	tio := NewTelnetIO(conn, s)

	return s, tio
}

func (s *streamingTty) Fd() uintptr {
	// You can return 0 or a dummy; v2 mainly uses this for termios on real TTYs.
	return 0
}

func (s *streamingTty) Read(p []byte) (int, error) {
	data, ok := <-s.inPipe
	if !ok {
		return 0, io.EOF
	}
	n := copy(p, data)
	return n, nil
}

func (s *streamingTty) Write(p []byte) (int, error) {
	buf := make([]byte, len(p))
	copy(buf, p)
	select {
	case s.outPipe <- buf:
		return len(p), nil
	default:
		return 0, io.ErrShortWrite
	}
}

func (s *streamingTty) Close() error {
	return s.Closer.Close()
}

// Window size & signals (you adapt to the exact v2 Tty API):

func (s *streamingTty) GetSize() (int, int, error) {
	s.rwMutex.Lock()
	defer s.rwMutex.Unlock()
	return s.width, s.height, nil
}

func (s *streamingTty) NotifyResize(f func()) {
	s.rwMutex.Lock()
	defer s.rwMutex.Unlock()
	s.onResize = f
}

func (s *streamingTty) SetSize(w, h int) {
	if w <= 0 || h <= 0 {
		return // ignore invalid sizes
	}
	s.rwMutex.Lock()
	changed := s.width != w || s.height != h
	s.width = w
	s.height = h
	s.rwMutex.Unlock()
	if changed {
		s.onResize() // Call the callback directly; it should be safe to do so.
	}
}

func (s *streamingTty) Drain() error {
	// Read all available input

	return nil
}

func (s *streamingTty) WindowSize() (WindowSize, error) {
	return WindowSize{Width: s.width, Height: s.height}, nil
}

func (s *streamingTty) Start() error {
	return nil
}

func (s *streamingTty) Stop() error {
	return s.Close()
}
