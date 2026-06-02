//go:build js && wasm

// Package session holds the authenticated player's state for the
// lifetime of the browser session. All scenes read from and write to this.
package session

import "connect4/client/net"

// State is the global player session, populated on login and cleared on sign-out.
type State struct {
	Token    string
	Username string
	ELO      int
	LoggedIn bool
}

// Current is the single session instance shared across all scenes.
var Current = &State{}

// CurrentWS holds the active WebSocket connection for the current match.
// Set by the matchmaking scene after a "matched" message; read by the game scene.
// Cleared by the result scene when the match is over.
var CurrentWS *net.WSConn

// Clear resets the session (sign-out).
func (s *State) Clear() {
	s.Token = ""
	s.Username = ""
	s.ELO = 0
	s.LoggedIn = false
}
