package clusterix

import (
	"errors"
	"fmt"
	"log"
	"net"
	"time"
)

type ConnectionListener interface {
	Connected(bool)
}

type DXListener interface {
	DX(DXMessage)
}

type WWVMessage struct{}

type WWVListener interface {
	WWV(WWVMessage)
}

type TextMessage struct{}

type TextListener interface {
	Text(TextMessage)
}

const (
	DefaultTimeout = time.Duration(5 * time.Second)
	DefaultPort    = 23 // telnet default port
)

var (
	ErrTimeout      = errors.New("timeout")
	ErrNotConnected = errors.New("not connected")
)

type Client struct {
	host     *net.TCPAddr
	username string
	password string
	trace    bool
	timeout  time.Duration

	outgoing       chan string
	closed         chan struct{}
	disconnectChan chan struct{}

	loggedIn bool

	listeners []any
}

func newClient(host *net.TCPAddr, username string, password string, trace bool) *Client {
	return &Client{
		host:     host,
		username: username,
		password: password,
		trace:    trace,
		timeout:  DefaultTimeout,

		outgoing: make(chan string),
		closed:   make(chan struct{}),
	}
}

func Open(host *net.TCPAddr, username string, password string, trace bool) (*Client, error) {
	client := newClient(host, username, password, trace)
	err := client.connect()
	if err != nil {
		return nil, err
	}

	return client, nil
}

func KeepOpen(host *net.TCPAddr, username string, password string, retryInterval time.Duration, trace bool) *Client {
	client := newClient(host, username, password, trace)

	go func() {
		disconnected := make(chan bool, 1)
		client.logTracef("connecting to %s...", host.IP.String())
		for {
			err := client.connect()
			if err == nil {
				client.WhenDisconnected(func() {
					disconnected <- true
				})
				select {
				case <-disconnected:
					client.logTracef("connection lost to %s, waiting for retry", host.IP.String())
				case <-client.closed:
					client.logTracef("connection closed")
					return
				}
			} else {
				client.logTracef("cannot connect to %s, waiting for retry: %v", host.IP.String(), err)
			}

			select {
			case <-time.After(retryInterval):
				client.logTracef("retrying to connect to %s", host.IP.String())
			case <-client.closed:
				client.logTracef("connection closed")
				return
			}
		}
	}()

	return client
}

func (c *Client) connect() error {
	if c.Connected() {
		return nil
	}

	host := c.host.IP.String()
	port := c.host.Port
	if port == 0 {
		port = DefaultPort
	}
	hostAddress := fmt.Sprintf("%s:%d", host, port)

	conn, err := net.Dial("tcp", hostAddress)
	if err != nil {
		return fmt.Errorf("cannot open TCP connection: %w", err)
	}
	c.disconnectChan = make(chan struct{})
	remoteAddr := conn.RemoteAddr()

	go c.readWriteLoop(conn)

	c.logTracef("connected to %s", remoteAddr.String())
	c.emitConnected(true)
	c.WhenDisconnected(func() {
		c.logTracef("disconnected from %s", remoteAddr.String())
		c.emitConnected(false)
	})

	return nil
}

func (c *Client) logTracef(format string, args ...any) {
	if c.trace {
		log.Printf(format, args...)
	}
}

func (c *Client) Connected() bool {
	if c.disconnectChan == nil {
		return false
	}
	select {
	case <-c.disconnectChan:
		return false
	default:
		return true
	}
}

func (c *Client) Disconnect() {
	// When the connection was disconnected from the outside, we keep it closed.
	select {
	case <-c.closed:
	default:
		close(c.closed)
	}

	if c.disconnectChan == nil {
		return
	}
	select {
	case <-c.disconnectChan:
		return
	default:
		close(c.disconnectChan)
	}
}

func (c *Client) WhenDisconnected(f func()) {
	if c.disconnectChan == nil {
		f()
		return
	}
	go func() {
		<-c.disconnectChan
		f()
	}()
}

func (c *Client) Notify(listener any) {
	c.listeners = append(c.listeners, listener)
}

func (c *Client) emitConnected(connected bool) {
	for _, l := range c.listeners {
		if listener, ok := l.(ConnectionListener); ok {
			listener.Connected(connected)
		}
	}
}

func (c *Client) emitDX(message DXMessage) {
	for _, listener := range c.listeners {
		if dxListener, ok := listener.(DXListener); ok {
			dxListener.DX(message)
		}
	}
}

func (c *Client) emitWWV(message WWVMessage) {
	for _, listener := range c.listeners {
		if wwvListener, ok := listener.(WWVListener); ok {
			wwvListener.WWV(message)
		}
	}
}

func (c *Client) emitText(message TextMessage) {
	for _, listener := range c.listeners {
		if textListener, ok := listener.(TextListener); ok {
			textListener.Text(message)
		}
	}
}
