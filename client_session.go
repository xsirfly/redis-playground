package main

import "time"

const (
	clientSessionTimeout = 1 * time.Hour
)

// ClientSession describes a client session.
type ClientSession struct {
	t *time.Timer
	c *Client
}

// StartSession start a client session
func StartSession(c *Client) *ClientSession {
	s := ClientSession{
		c: c,
		t: time.NewTimer(clientSessionTimeout),
	}

	// Wait for timer to elapse.
	go func() {
		<-s.t.C
		c.send <- []byte("Session ended.")
		// Sleep some before closing client connection.
		time.Sleep(5 * time.Second)
		c.conn.Close()
	}()
	return &s
}
