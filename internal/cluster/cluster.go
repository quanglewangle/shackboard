// Package cluster maintains a persistent connection to a DX cluster telnet
// node and buffers the spots it streams.
package cluster

import (
	"bufio"
	"context"
	"fmt"
	"log"
	"net"
	"os"
	"strings"
	"sync/atomic"
	"time"
)

// Client connects to a single DX cluster node, logs in with the configured
// callsign, and feeds parsed spots into Buffer for as long as ctx is alive.
type Client struct {
	Host   string
	Call   string
	Buffer *Buffer

	connected atomic.Bool
	debug     bool
}

func NewClient(host, call string, buf *Buffer) *Client {
	return &Client{
		Host:   host,
		Call:   call,
		Buffer: buf,
		debug:  os.Getenv("SHACKBOARD_DEBUG") != "",
	}
}

func (c *Client) Connected() bool {
	return c.connected.Load()
}

// Run is the top-level reconnect/backoff loop. Call once in a goroutine with
// a context tied to server lifetime.
func (c *Client) Run(ctx context.Context) {
	backoff := 5 * time.Second
	const maxBackoff = 5 * time.Minute

	for {
		if ctx.Err() != nil {
			return
		}

		err := c.connectOnce(ctx)
		established := c.connected.Load()
		c.connected.Store(false)

		if ctx.Err() != nil {
			return
		}

		if err != nil {
			log.Printf("cluster: %s: %v (retrying in %s)", c.Host, err, backoff)
		}

		if established {
			// Got past login at least once this session — don't punish a
			// long-lived, well-behaved connection with a slow reconnect.
			backoff = 5 * time.Second
		}

		select {
		case <-ctx.Done():
			return
		case <-time.After(backoff):
		}

		if !established {
			backoff *= 2
			if backoff > maxBackoff {
				backoff = maxBackoff
			}
		}
	}
}

func (c *Client) connectOnce(ctx context.Context) error {
	dialer := net.Dialer{Timeout: 15 * time.Second}
	conn, err := dialer.DialContext(ctx, "tcp", c.Host)
	if err != nil {
		return fmt.Errorf("dial: %w", err)
	}
	defer conn.Close()

	done := make(chan struct{})
	defer close(done)
	go func() {
		select {
		case <-ctx.Done():
			conn.Close()
		case <-done:
		}
	}()

	if err := c.login(conn); err != nil {
		return fmt.Errorf("login: %w", err)
	}

	c.connected.Store(true)
	return c.readSpots(conn)
}

// login waits for the cluster's login prompt, which is known (from a live
// capture against dxc.ve7cc.net) to arrive without a trailing newline, e.g.
// "Please enter your call: " followed by "login: " — so it can't be read
// with a line-oriented scanner. Reads byte by byte until the accumulated
// text ends with "login:" or "call:", then sends the configured callsign.
func (c *Client) login(conn net.Conn) error {
	conn.SetReadDeadline(time.Now().Add(30 * time.Second))

	var acc strings.Builder
	buf := make([]byte, 1)
	for {
		n, err := conn.Read(buf)
		if n > 0 {
			acc.WriteByte(buf[0])
		}
		if err != nil {
			return fmt.Errorf("waiting for login prompt: %w", err)
		}

		lower := strings.ToLower(acc.String())
		if strings.HasSuffix(lower, "login:") || strings.HasSuffix(lower, "call:") {
			_, err := conn.Write([]byte(c.Call + "\r\n"))
			return err
		}

		if acc.Len() > 4096 {
			return fmt.Errorf("no login prompt seen in first 4096 bytes")
		}
	}
}

// readSpots reads line by line until the connection dies or ctx cancels it,
// parsing and buffering any spot lines. Most lines aren't spots (command
// prompts, WWV announcements, the propagation table) — those are silently
// skipped, only logged when SHACKBOARD_DEBUG is set.
func (c *Client) readSpots(conn net.Conn) error {
	scanner := bufio.NewScanner(conn)
	scanner.Buffer(make([]byte, 4096), 64*1024)

	for {
		conn.SetReadDeadline(time.Now().Add(10 * time.Minute))
		if !scanner.Scan() {
			if err := scanner.Err(); err != nil {
				return fmt.Errorf("read: %w", err)
			}
			return fmt.Errorf("connection closed")
		}

		line := scanner.Text()
		spot, ok := parseSpotLine(line)
		if !ok {
			if c.debug && strings.TrimSpace(line) != "" {
				log.Printf("cluster: unmatched line: %q", line)
			}
			continue
		}
		c.Buffer.Add(spot)
	}
}
