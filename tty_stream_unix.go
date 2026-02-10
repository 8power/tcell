//go:build aix || darwin || dragonfly || freebsd || linux || netbsd || openbsd || solaris || zos
// +build aix darwin dragonfly freebsd linux netbsd openbsd solaris zos

package tcell

import (
	"bytes"
	"fmt"
	"io"
	"net"
	"sync"
)

// Telnet IAC codes for NAWS (Negotiate About Window Size)
// https://en.wikipedia.org/wiki/Telnet
const (
	IAC  = 255 // Sequence Initializer/Escape Character
	SB   = 250 // Initiate the negotiation of a sub-service of a protocol mechanism
	SE   = 240 // End of subnegotiation parameters
	WILL = 251 // Informs other party that this party will use a protocol mechanism
	WONT = 252 // Informs other party that this party will not use a protocol mechanism
	DO   = 253 // Instruct other party to use a protocol mechanism
	DONT = 254 // Instruct other party to not use a protocol mechanism
	NAWS = 31  // Telnet option code for NAWS (Negotiate About Window Size)
)

type streamingTty struct {
	io.ReadWriteCloser            // e.g. SSH channel
	mtx                sync.Mutex // mutex to protect access to rw
	width              int        // character width handling
	height             int        // character height handling
	onResize           func()     // callback for resize events
}

func NewStreamingTty(conn net.Conn) Tty {
	s := &streamingTty{
		ReadWriteCloser: conn,
		width:           80,
		height:          24,
	}

	// Ask client to send NAWS
	s.mtx.Lock()
	s.ReadWriteCloser.Write([]byte{IAC, DO, NAWS})
	s.mtx.Unlock()
	return s
}

func (s *streamingTty) Fd() uintptr {
	// You can return 0 or a dummy; v2 mainly uses this for termios on real TTYs.
	return 0
}

func (s *streamingTty) Read(p []byte) (int, error) {
	n, err := s.ReadWriteCloser.Read(p)
	if err != nil && err != io.EOF {
		return n, err
	}
	if n > 0 {
		// Parse data for telent NAWS protocol
		parsedData := s.parseNAWS(p)
		copy(p, parsedData)
		n = len(parsedData)
	}
	return n, err
}

func (s *streamingTty) parseNAWS(p []byte) []byte {
	parsedBuffer := make([]byte, 0)
	// read p as a stream of bytes, looking for IAC sequences
	reader := bytes.NewReader(p)
	for {
		b, err := reader.ReadByte()
		if err != nil {
			break // EOF or error
		}
		if b == IAC {
			// handle IAC sequence (e.g. NAWS)
			// read next bytes to determine command and option
			cmd, err := reader.ReadByte()
			if err != nil {
				break
			}
			switch cmd {
			case SB:
				opt, err := reader.ReadByte()
				if err != nil {
					break
				}
				if opt == NAWS {
					// read width and height
					wh := make([]byte, 4)
					if _, err := io.ReadFull(reader, wh); err != nil {
						break
					}
					width := int(wh[0])<<8 | int(wh[1])
					height := int(wh[2])<<8 | int(wh[3])
					s.SetSize(width, height)
					// consume IAC SE
					if _, err := reader.ReadByte(); err != nil {
						break
					}
					if _, err := reader.ReadByte(); err != nil {
						break
					}
				} else {
					// skip unknown SB option until IAC SE
					for {
						b, err := reader.ReadByte()
						if err != nil {
							break
						}
						if b == IAC {
							next, err := reader.ReadByte()
							if err != nil {
								break
							}
							if next == SE {
								break // end of subnegotiation
							}
						}
					}
				}
			case WILL, WONT, DO, DONT:
				// read option byte
				opt, err := reader.ReadByte()
				if err != nil {
					break
				}
				switch cmd {
				case WILL:
					if opt == NAWS {
						// client will send NAWS, fine
						return nil
					}
					// decline other options
					s.mtx.Lock()
					s.ReadWriteCloser.Write([]byte{IAC, DONT, opt})
					s.mtx.Unlock()
				case DO:
					// We won't do anything
					s.mtx.Lock()
					s.ReadWriteCloser.Write([]byte{IAC, WONT, opt})
					s.mtx.Unlock()
				}
			default:
				// ignore other commands for now
				continue

			}
		} else {
			// normal data byte, add to parsed buffer
			parsedBuffer = append(parsedBuffer, b)
		}
	}
	return parsedBuffer
}

func (s *streamingTty) Write(p []byte) (int, error) {
	// Escape IAC bytes in output
	escapedData := make([]byte, 0, len(p)*2)
	for _, b := range p {
		if b == IAC {
			escapedData = append(escapedData, IAC) // double IAC for escaping
		}
		escapedData = append(escapedData, b)
	}

	s.mtx.Lock()
	defer s.mtx.Unlock()
	n, err := s.ReadWriteCloser.Write(escapedData)
	if err != nil {
		return n, fmt.Errorf("failed to write to connection: %w", err)
	}
	return n, nil
}

func (s *streamingTty) Close() error {
	return s.ReadWriteCloser.Close()
}

// Window size & signals (you adapt to the exact v2 Tty API):

func (s *streamingTty) GetSize() (int, int, error) {
	s.mtx.Lock()
	defer s.mtx.Unlock()
	return s.width, s.height, nil
}

func (s *streamingTty) NotifyResize(f func()) {
	s.mtx.Lock()
	defer s.mtx.Unlock()
	s.onResize = f
}

func (s *streamingTty) SetSize(w, h int) {
	if w <= 0 || h <= 0 {
		return // ignore invalid sizes
	}
	s.mtx.Lock()
	changed := s.width != w || s.height != h
	s.width = w
	s.height = h
	s.mtx.Unlock()
	if changed {
		if s.onResize != nil {
			s.onResize() // Call the callback directly; it should be safe to do so.
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
	return s.ReadWriteCloser.Close()
}
