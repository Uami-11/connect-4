//go:build js && wasm

package net

import (
	"fmt"
	"strings"
	"syscall/js"
)

// WSConn is a thin wrapper around the browser WebSocket API.
type WSConn struct {
	ws    js.Value
	recv  chan string
	done  chan struct{}
}

// NewWSConn opens a WebSocket connection to /ws on the current origin.
// Returns once the connection is open or returns an error if it fails.
func NewWSConn() (*WSConn, error) {
	origin := js.Global().Get("location").Get("origin").String()
	wsURL := strings.Replace(origin, "http", "ws", 1) + "/ws"

	c := &WSConn{
		recv: make(chan string, 32),
		done: make(chan struct{}),
	}

	opened := make(chan error, 1)

	ws := js.Global().Get("WebSocket").New(wsURL)
	c.ws = ws

	ws.Set("onopen", js.FuncOf(func(_ js.Value, _ []js.Value) any {
		opened <- nil
		return nil
	}))
	ws.Set("onerror", js.FuncOf(func(_ js.Value, args []js.Value) any {
		opened <- fmt.Errorf("websocket error")
		return nil
	}))
	ws.Set("onmessage", js.FuncOf(func(_ js.Value, args []js.Value) any {
		data := args[0].Get("data").String()
		select {
		case c.recv <- data:
		default:
			// Drop if buffer full — should not happen in normal play.
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

	if err := <-opened; err != nil {
		return nil, err
	}
	return c, nil
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
