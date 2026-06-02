//go:build js && wasm

package net

import (
	"strings"
	"syscall/js"
)

// WSConn is a thin wrapper around the browser WebSocket API.
type WSConn struct {
	ws   js.Value
	recv chan string
	done chan struct{}
}

// NewWSConn opens a WebSocket connection to /ws on the current origin.
// Returns immediately without waiting for the connection to open.
// Call IsOpen() to poll the connection state.
func NewWSConn() (*WSConn, error) {
	origin := js.Global().Get("location").Get("origin").String()
	wsURL := strings.Replace(origin, "http", "ws", 1) + "/ws"

	c := &WSConn{
		recv: make(chan string, 32),
		done: make(chan struct{}),
	}

	ws := js.Global().Get("WebSocket").New(wsURL)
	c.ws = ws

	ws.Set("onmessage", js.FuncOf(func(_ js.Value, args []js.Value) any {
		data := args[0].Get("data").String()
		select {
		case c.recv <- data:
		default:
		}
		return nil
	}))
	ws.Set("onclose", js.FuncOf(func(_ js.Value, _ []js.Value) any {
		select {
		case <-c.done:
		default:
			close(c.done)
		}
		return nil
	}))

	return c, nil
}

// IsOpen returns true when the WebSocket connection is open and ready.
func (c *WSConn) IsOpen() bool {
	return c.ws.Get("readyState").Int() == 1 // WebSocket.OPEN
}

// Send sends a raw JSON string over the WebSocket.
func (c *WSConn) Send(msg string) {
	c.ws.Call("send", msg)
}

// Recv returns the channel that delivers incoming messages.
func (c *WSConn) Recv() <-chan string {
	return c.recv
}

// Done returns a channel closed when the connection is lost.
func (c *WSConn) Done() <-chan struct{} {
	return c.done
}

// Close closes the WebSocket.
func (c *WSConn) Close() {
	c.ws.Call("close")
}
