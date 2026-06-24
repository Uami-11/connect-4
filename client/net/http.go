//go:build js && wasm

package net

import (
	"encoding/json"
	"fmt"
	"strings"
	"syscall/js"
)

type PendingRequest struct {
	req  js.Value
	Done bool
	err  error
	text string
}

func StartPost(path string, body any) *PendingRequest {
	b, err := json.Marshal(body)
	if err != nil {
		return &PendingRequest{Done: true, err: fmt.Errorf("marshalling body: %w", err)}
	}

	req := js.Global().Get("XMLHttpRequest").New()
	url := js.Global().Get("location").Get("origin").String() + path
	req.Call("open", "POST", url, true)
	req.Call("setRequestHeader", "Content-Type", "application/json")
	req.Call("send", string(b))
	return &PendingRequest{req: req}
}

// StartPostAuth is like StartPost but includes a Bearer token.
func StartPostAuth(path string, body any, token string) *PendingRequest {
	b, err := json.Marshal(body)
	if err != nil {
		return &PendingRequest{Done: true, err: fmt.Errorf("marshalling body: %w", err)}
	}
	req := js.Global().Get("XMLHttpRequest").New()
	url := js.Global().Get("location").Get("origin").String() + path
	req.Call("open", "POST", url, true)
	req.Call("setRequestHeader", "Content-Type", "application/json")
	if token != "" {
		req.Call("setRequestHeader", "Authorization", "Bearer "+token)
	}
	req.Call("send", string(b))
	return &PendingRequest{req: req}
}

func StartGet(path, token string) *PendingRequest {
	req := js.Global().Get("XMLHttpRequest").New()
	url := js.Global().Get("location").Get("origin").String() + path
	req.Call("open", "GET", url, true)
	if token != "" {
		req.Call("setRequestHeader", "Authorization", "Bearer "+token)
	}
	req.Call("send")
	return &PendingRequest{req: req}
}

func Poll(pr *PendingRequest) {
	if pr == nil || pr.Done {
		return
	}
	if pr.req.Get("readyState").Int() != 4 {
		return
	}
	pr.Done = true
	status := pr.req.Get("status").Int()
	if status >= 200 && status < 300 {
		pr.text = pr.req.Get("responseText").String()
	} else {
		body := pr.req.Get("responseText").String()
		if body == "" {
			body = "unknown error"
		}
		pr.err = fmt.Errorf("server error %d: %s", status, body)
	}
}

// Err returns any error that occurred during the request.
func (pr *PendingRequest) Err() error {
	return pr.err
}

func DecodeResult(pr *PendingRequest, result any) error {
	if pr.err != nil {
		return pr.err
	}
	if result != nil {
		return json.NewDecoder(strings.NewReader(pr.text)).Decode(result)
	}
	return nil
}
