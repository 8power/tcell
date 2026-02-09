//go:build aix || darwin || dragonfly || freebsd || linux || netbsd || openbsd || solaris || zos
// +build aix darwin dragonfly freebsd linux netbsd openbsd solaris zos

// This sits inbetween a telnet connection and the tcell tty.
// It handles the telnet protocol, and translates it into a streaming Tty that tcell can use.
// This allows tcell to be used in a telnet server context, without needing to worry about the details of the telnet protocol.
// The main thing it handles is NAWS (Negotiate About Window Size), which allows the client to inform us of the terminal dimensions.
// It also handles basic option negotiation, but for simplicity it only accepts NAWS and declines all other options.
// This is sufficient for most telnet clients, which typically support NAWS and will fall back gracefully if it's not accepted.

package tcell

import (
	"fmt"
	"io"
	"net"
	"sync"
	"time"
)

const (
	IAC  = 255
	SB   = 250
	SE   = 240
	WILL = 251
	WONT = 252
	DO   = 253
	DONT = 254
	NAWS = 31
)

type TelnetIO struct {
	conn net.Conn

	inPipe  chan []byte // to tcell (input)
	outPipe chan []byte // from tcell (output)

	tty *streamingTty

	mtx sync.Mutex // protects access to the connection for writing
}

func NewTelnetIO(conn net.Conn, tty *streamingTty) *TelnetIO {
	t := &TelnetIO{
		conn:    conn,
		inPipe:  tty.inPipe,
		outPipe: tty.outPipe,
		tty:     tty,
	}
	go t.readLoop()
	go t.writeLoop()
	return t
}

func (t *TelnetIO) readLoop() {
	defer close(t.inPipe)

	buf := make([]byte, 1)

	t.mtx.Lock()
	// Ask client to send NAWS
	t.conn.Write([]byte{IAC, DO, NAWS})
	t.mtx.Unlock()

	for {
		// reads 1 byte at a time to handle IAC sequences properly
		n, err := t.conn.Read(buf)
		if err != nil || n == 0 {
			return
		}
		b := buf[0]

		if b == IAC {
			// flush buffered normal input before handling control
			if err := t.handleIAC(); err != nil {
				return
			}
		} else {
			select {
			case t.inPipe <- []byte{b}:
				// sent byte to tcell
			default:
				// if tcell is not reading, drop input to avoid blocking
			}
		}
	}
}

func (t *TelnetIO) handleIAC() error {
	b := []byte{0}
	if _, err := t.conn.Read(b); err != nil {
		return err
	}
	cmd := b[0]

	switch cmd {
	case SB:
		return t.handleSubnegotiation()
	case WILL, WONT, DO, DONT:
		// read option byte
		if _, err := t.conn.Read(b); err != nil {
			return err
		}
		opt := b[0]
		// very simple negotiation; accept NAWS, decline others
		switch cmd {
		case WILL:
			if opt == NAWS {
				// client will send NAWS, fine
				return nil
			}
			// decline other options
			t.mtx.Lock()
			t.conn.Write([]byte{IAC, DONT, opt})
			t.mtx.Unlock()
		case DO:
			if opt == NAWS {
				// we won’t send NAWS to client
				t.mtx.Lock()
				t.conn.Write([]byte{IAC, WONT, NAWS})
				t.mtx.Unlock()
				return nil
			}
			t.mtx.Lock()
			t.conn.Write([]byte{IAC, WONT, opt})
			t.mtx.Unlock()
		}
	default:
		// for simplicity, ignore other commands
	}
	return nil
}

func (t *TelnetIO) handleSubnegotiation() error {
	b := []byte{0}

	// which option?
	if _, err := t.conn.Read(b); err != nil {
		return err
	}
	opt := b[0]

	if opt == NAWS {
		wh := make([]byte, 4)
		if _, err := io.ReadFull(t.conn, wh); err != nil {
			return err
		}
		width := int(wh[0])<<8 | int(wh[1])
		height := int(wh[2])<<8 | int(wh[3])

		// consume IAC SE
		if _, err := t.conn.Read(b); err != nil {
			return err
		}
		if b[0] != IAC {
			return fmt.Errorf("expected IAC after NAWS data")
		}
		if _, err := t.conn.Read(b); err != nil {
			return err
		}
		if b[0] != SE {
			return fmt.Errorf("expected SE after NAWS")
		}

		t.tty.SetSize(width, height)
		return nil
	}

	// skip unknown SB until IAC SE
	for {
		if _, err := t.conn.Read(b); err != nil {
			return err
		}
		if b[0] == IAC {
			if _, err := t.conn.Read(b); err != nil {
				return err
			}
			if b[0] == SE {
				return nil
			}
		}
	}
}

func (t *TelnetIO) writeLoop() {
	defer t.conn.Close()

	for data := range t.outPipe {
		var escapedData []byte
		for _, b := range data {
			escapedData = append(escapedData, b)
			if b == IAC {
				// Escape IAC by doubling it
				escapedData = append(escapedData, IAC)
			}
		}
		t.mtx.Lock()
		err := t.conn.SetWriteDeadline(time.Now().Add(5 * time.Second))
		if err != nil {
			t.mtx.Unlock()
			return
		}
		_, err = t.conn.Write(escapedData)
		t.mtx.Unlock()
		if err != nil {
			return
		}
	}
}
