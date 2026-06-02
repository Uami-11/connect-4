//go:build js && wasm

// Package net provides HTTP and WebSocket client helpers for the WASM client.
// All calls use the browser's fetch and WebSocket APIs via syscall/js.
package net

import (
	"encoding/json"
	"fmt"
	"strings"
	"syscall/js"
)

// Post sends a JSON POST request to the given path and returns the decoded body.
// The path is relative to the current origin (e.g. "/login").
func Post(path string, body any, result any) error {
	b, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("marshalling body: %w", err)
	}

	ch := make(chan error, 1)
	var responseText string

	// Build fetch options.
	opts := js.Global().Get("Object").New()
	opts.Set("method", "POST")
	headers := js.Global().Get("Object").New()
	headers.Set("Content-Type", "application/json")
	opts.Set("headers", headers)
	opts.Set("body", string(b))

	url := js.Global().Get("location").Get("origin").String() + path

	fetch := js.Global().Get("fetch").Invoke(url, opts)
	fetch.Call("then", js.FuncOf(func(this js.Value, args []js.Value) any {
		resp := args[0]
		if resp.Get("ok").Bool() {
			resp.Call("text").Call("then", js.FuncOf(func(_ js.Value, args []js.Value) any {
				responseText = args[0].String()
				ch <- nil
				return nil
			}))
		} else {
			resp.Call("text").Call("then", js.FuncOf(func(_ js.Value, args []js.Value) any {
				ch <- fmt.Errorf("server error %d: %s", resp.Get("status").Int(), args[0].String())
				return nil
			}))
		}
		return nil
	})).Call("catch", js.FuncOf(func(_ js.Value, args []js.Value) any {
		ch <- fmt.Errorf("network error: %s", args[0].Get("message").String())
		return nil
	}))

	err = <-ch
	if err != nil {
		return err
	}
	if result != nil {
		return json.NewDecoder(strings.NewReader(responseText)).Decode(result)
	}
	return nil
}

// Get sends a GET request and decodes the JSON response into result.
func Get(path, token string, result any) error {
	ch := make(chan error, 1)
	var responseText string

	opts := js.Global().Get("Object").New()
	opts.Set("method", "GET")
	if token != "" {
		headers := js.Global().Get("Object").New()
		headers.Set("Authorization", "Bearer "+token)
		opts.Set("headers", headers)
	}

	url := js.Global().Get("location").Get("origin").String() + path

	fetch := js.Global().Get("fetch").Invoke(url, opts)
	fetch.Call("then", js.FuncOf(func(_ js.Value, args []js.Value) any {
		resp := args[0]
		if resp.Get("ok").Bool() {
			resp.Call("text").Call("then", js.FuncOf(func(_ js.Value, args []js.Value) any {
				responseText = args[0].String()
				ch <- nil
				return nil
			}))
		} else {
			resp.Call("text").Call("then", js.FuncOf(func(_ js.Value, args []js.Value) any {
				ch <- fmt.Errorf("server error %d: %s", resp.Get("status").Int(), args[0].String())
				return nil
			}))
		}
		return nil
	})).Call("catch", js.FuncOf(func(_ js.Value, args []js.Value) any {
		ch <- fmt.Errorf("network error: %s", args[0].Get("message").String())
		return nil
	}))

	err := <-ch
	if err != nil {
		return err
	}
	if result != nil {
		return json.NewDecoder(strings.NewReader(responseText)).Decode(result)
	}
	return nil
}
