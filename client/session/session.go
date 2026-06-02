//go:build js && wasm

// Package session holds the authenticated player's state for the
// lifetime of the browser session. All scenes read from and write to this.
package session

// State is the global player session, populated on login and cleared on sign-out.
type State struct {
	Token    string
	Username string
	ELO      int
	LoggedIn bool
}

// Current is the single session instance shared across all scenes.
var Current = &State{}

// Clear resets the session (sign-out).
func (s *State) Clear() {
	s.Token = ""
	s.Username = ""
	s.ELO = 0
	s.LoggedIn = false
}
