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

// CurrentMatchColor holds the player's color (1=red, 2=yellow) for the current match.
var CurrentMatchColor int

// CurrentMatchOpponent holds the opponent's username.
var CurrentMatchOpponent string

// CurrentWS holds the active WebSocket connection for the current match.
// Set by the matchmaking scene after a "matched" message; read by the game scene.
// Cleared by the result scene when the match is over.
var CurrentWS *net.WSConn

// GameResult carries end-of-match data from game scene to result scene.
type GameResult struct {
	Outcome   string
	WinColor  string
	ELOBefore int
	ELOAfter  int
	ELODelta  int
}

// CurrentResult is set by the game scene before navigating to IDResult.
// Cleared when the result scene is done.
var CurrentResult *GameResult

// CurrentOtherUsername holds the username to view in ProfileOther.
// Set before navigating to IDProfileOther.
var CurrentOtherUsername string

// Clear resets the session (sign-out).
func (s *State) Clear() {
	s.Token = ""
	s.Username = ""
	s.ELO = 0
	s.LoggedIn = false
}
