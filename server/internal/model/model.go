package model

import "time"

// User is a registered player stored in the database.
type User struct {
	ID           int       `json:"id"`
	Username     string    `json:"username"`
	PasswordHash string    `json:"-"`
	ELO          int       `json:"elo"`
	CreatedAt    time.Time `json:"created_at"`
	LastActiveAt *time.Time `json:"last_active_at"`
}

// Match is a completed game stored in the database.
type Match struct {
	ID              int       `json:"id"`
	Player1ID       int       `json:"player1_id"`
	Player2ID       int       `json:"player2_id"`
	WinnerID        *int      `json:"winner_id"` // nil = draw
	Player1ELOBefore int      `json:"player1_elo_before"`
	Player2ELOBefore int      `json:"player2_elo_before"`
	ELODelta        int       `json:"elo_delta"`
	PlayedAt        time.Time `json:"played_at"`
}

// MatchHistoryEntry is one row in a player's match history view.
type MatchHistoryEntry struct {
	MatchID      int       `json:"match_id"`
	OpponentName string    `json:"opponent_name"`
	Outcome      string    `json:"outcome"`  // "win", "loss", "draw"
	ELOBefore    int       `json:"elo_before"`
	ELOAfter     int       `json:"elo_after"`
	ELODelta     int       `json:"elo_delta"` // positive = gained, negative = lost
	PlayedAt     time.Time `json:"played_at"`
}

// LeaderboardEntry is one row in the global leaderboard.
type LeaderboardEntry struct {
	Rank     int    `json:"rank"`
	Username string `json:"username"`
	ELO      int    `json:"elo"`
	Wins     int    `json:"wins"`
	Losses   int    `json:"losses"`
	Draws    int    `json:"draws"`
}

// PublicProfile is a player's profile visible to any user.
type PublicProfile struct {
	Username     string              `json:"username"`
	ELO          int                 `json:"elo"`
	Wins         int                 `json:"wins"`
	Losses       int                 `json:"losses"`
	Draws        int                 `json:"draws"`
	CreatedAt    time.Time           `json:"created_at"`
	LastActiveAt *time.Time          `json:"last_active_at"`
	Online       bool                `json:"online"`
	History      []MatchHistoryEntry `json:"history"`
}

// WSMessage is the envelope for every WebSocket message.
type WSMessage struct {
	Type    string `json:"type"`
	Payload any    `json:"payload,omitempty"`
}

// Outbound payload types — server → client.

type MatchedPayload struct {
	OpponentName string `json:"opponent_name"`
	YourColor    string `json:"your_color"` // "red" or "yellow"
	YourTurn     bool   `json:"your_turn"`
}

type StatePayload struct {
	Board    [6][7]int `json:"board"`
	Turn     int       `json:"turn"` // 1 = red, 2 = yellow
}

type ResultPayload struct {
	Outcome   string `json:"outcome"`    // "win", "loss", "draw"
	WinColor  string `json:"win_color"`  // "red", "yellow", or "draw"
	ELOBefore int    `json:"elo_before"`
	ELOAfter  int    `json:"elo_after"`
	ELODelta  int    `json:"elo_delta"`
}

type DisconnectPayload struct {
	SecondsRemaining int `json:"seconds_remaining"`
}

// Inbound payload types — client → server.

type PlacePayload struct {
	Column int `json:"column"`
}

type RejoinPayload struct {
	Token string `json:"token"`
}
