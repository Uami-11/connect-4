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

// PendingChallenge represents an incoming or accepted challenge.
type PendingChallenge struct {
	FromUsername string // who sent it
	Status       string // "pending", "accepted"
	// For accepted challenges, used to navigate to game.
	OpponentName string
	YourColor    string
	YourTurn     bool
}

// CurrentChallenge is set by the polling loop in Manager.Update when
// a pending challenge is detected. Cleared when the user acts on it.
var CurrentChallenge *PendingChallenge

// ClearChallenge resets the current challenge state.
func ClearChallenge() {
	CurrentChallenge = nil
}

// CurrentOtherUsername holds the username to view in ProfileOther.
// Set before navigating to IDProfileOther.
var CurrentOtherUsername string

// Save persists the session to localStorage.
func (s *State) Save() {
	saveSession(s.Token, s.Username, s.ELO)
}

// LoadSession reads saved credentials from localStorage and populates Current.
// Returns true if a saved session was found.
func LoadSession() bool {
	token, username, elo, ok := loadSession()
	if !ok {
		return false
	}
	Current.Token = token
	Current.Username = username
	Current.ELO = elo
	Current.LoggedIn = true
	return true
}

// Clear resets the session (sign-out).
func (s *State) Clear() {
	s.Token = ""
	s.Username = ""
	s.ELO = 0
	s.LoggedIn = false
	clearSession()
}
