//go:build aix || darwin || dragonfly || freebsd || linux || netbsd || openbsd || solaris || zos
// +build aix darwin dragonfly freebsd linux netbsd openbsd solaris zos

package tcell

import (
	"io"
	"sync"
)

type streamingTty struct {
	rw       io.ReadWriter // e.g. SSH channel
	rwMutex  sync.Mutex    // mutex to protect access to rw
	width    int           // character width handling
	height   int           // character height handling
	resizeQ  chan struct{} // channel to signal resize events
	onResize func()        // callback for resize events
}

func NewStreamingTty(rw io.ReadWriter) Tty {

	s := &streamingTty{
		rw:      rw,
		width:   80,
		height:  24,
		resizeQ: make(chan struct{}, 1), // buffered to avoid blocking
	}
	// Background goroutine to watch for resizes and invoke the callback
	go s.resizeWatcher()
	return s
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
		// Signal the resize event (non-blocking)
		select {
		case s.resizeQ <- struct{}{}:
		default:
		}
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
	close(s.resizeQ)
	return nil
}

// Internal: watches resizeQ and invokes the registered callback.
func (s *streamingTty) resizeWatcher() {
	for range s.resizeQ {
		s.rwMutex.Lock()
		cb := s.onResize
		s.rwMutex.Unlock()
		if cb != nil {
			cb()
		}
	}
}
