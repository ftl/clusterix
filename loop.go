package clusterix

import (
	"errors"
	"io"
	"net"
	"os"
	"strings"
	"time"
)

func (c *Client) readWriteLoop(conn net.Conn) {
	defer conn.Close()

	reader := newConnReader(conn, 10*1024, 100*time.Millisecond)

	for {
		select {
		case <-c.disconnectChan:
			return
		default:
			text, err := reader.read()
			if err != nil {
				if errors.Is(err, io.EOF) {
					c.logTracef("connection closed")
					close(c.disconnectChan)
					return
				} else {
					c.logTracef("cannot read incoming data: %v", err)
					close(c.disconnectChan)
					return
				}
			}

			response := c.handleIncomingText(text)

			if response != "" {
				c.logTracef("< %q", response)
				_, err := conn.Write([]byte(response))
				if err != nil {
					c.logTracef("cannot write outgoing data: %v", err)
				}
			}
		}
	}
}

func (c *Client) handleIncomingText(text string) string {
	text = strings.ToLower(strings.TrimSpace(text))
	if text == "" {
		return ""
	}
	c.logTracef("> %q", text)

	switch {
	case !c.loggedIn && hasAnySuffix(text, "callsign:", "call:", "login:"):
		return c.username + "\r\n"
	case !c.loggedIn && hasAnySuffix(text, "password:"):
		return c.password + "\r\n"
	case !c.loggedIn && hasAnySuffix(text, "z cwskimmer >", "z dxspider >", "z arc6>", "ccc >"):
		c.loggedIn = true
		return ""
	}

	dxMessages, err := ExtractDXMessages(text)
	if err != nil {
		c.logTracef("cannot extract DX messages: %v", err)
		return ""
	}
	for _, msg := range dxMessages {
		c.emitDX(msg)
	}

	return ""
}

func hasAnySuffix(s string, suffixes ...string) bool {
	for _, suffix := range suffixes {
		if strings.HasSuffix(s, suffix) {
			return true
		}
	}
	return false
}

type connReader struct {
	conn        net.Conn
	readBuffer  []byte
	readTimeout time.Duration
}

func newConnReader(conn net.Conn, bufferSize int, readTimeout time.Duration) *connReader {
	return &connReader{
		conn:        conn,
		readBuffer:  make([]byte, bufferSize),
		readTimeout: readTimeout,
	}
}

func (r *connReader) read() (string, error) {
	result := ""
	for {
		nextReadDeadline := time.Now().Add(r.readTimeout)
		r.conn.SetReadDeadline(nextReadDeadline)
		n, err := r.conn.Read(r.readBuffer)
		if err != nil {
			switch {
			case errors.Is(err, os.ErrDeadlineExceeded):
				return result, nil
			default:
				return "", err
			}
		}
		result += string(r.readBuffer[:n])
	}
}
