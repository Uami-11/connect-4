//go:build js && wasm

package session

import (
	"strconv"
	"syscall/js"
)

func saveSession(token, username string, elo int) {
	ls := js.Global().Get("localStorage")
	ls.Call("setItem", "connect4_token", token)
	ls.Call("setItem", "connect4_username", username)
	ls.Call("setItem", "connect4_elo", strconv.Itoa(elo))
}

func loadSession() (token, username string, elo int, ok bool) {
	ls := js.Global().Get("localStorage")

	v := ls.Call("getItem", "connect4_token")
	if v.IsNull() || v.IsUndefined() || v.String() == "" {
		return "", "", 0, false
	}
	token = v.String()

	v = ls.Call("getItem", "connect4_username")
	if v.IsNull() || v.IsUndefined() {
		return "", "", 0, false
	}
	username = v.String()

	v = ls.Call("getItem", "connect4_elo")
	if !v.IsNull() && !v.IsUndefined() {
		elo, _ = strconv.Atoi(v.String())
	}

	return token, username, elo, true
}

func clearSession() {
	ls := js.Global().Get("localStorage")
	ls.Call("removeItem", "connect4_token")
	ls.Call("removeItem", "connect4_username")
	ls.Call("removeItem", "connect4_elo")
}
