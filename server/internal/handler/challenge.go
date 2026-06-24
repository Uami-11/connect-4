package handler

import (
	"encoding/json"
	"net/http"
	"strings"

	"connect4/server/internal/auth"
	"connect4/server/internal/game"
	"connect4/server/internal/model"
)

// ChallengeHTTP handles challenge-related HTTP endpoints.
type ChallengeHTTP struct {
	cm        *game.ChallengeManager
	mm        *game.Matchmaker
	jwtSecret string
}

// NewChallengeHTTP creates a ChallengeHTTP handler.
func NewChallengeHTTP(cm *game.ChallengeManager, mm *game.Matchmaker, jwtSecret string) *ChallengeHTTP {
	return &ChallengeHTTP{cm: cm, mm: mm, jwtSecret: jwtSecret}
}

// userFromRequest extracts user info from a Bearer token in the Authorization header.
func (h *ChallengeHTTP) userFromRequest(r *http.Request) (*auth.Claims, error) {
	ah := r.Header.Get("Authorization")
	if !strings.HasPrefix(ah, "Bearer ") {
		return nil, http.ErrNoCookie
	}
	return auth.ParseToken(ah[7:], h.jwtSecret)
}

// Send handles POST /challenge/send.
// Request body: {"target_username": "..."}
func (h *ChallengeHTTP) Send(w http.ResponseWriter, r *http.Request) {
	claims, err := h.userFromRequest(r)
	if err != nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	var body struct {
		TargetUsername string `json:"target_username"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}
	body.TargetUsername = strings.TrimSpace(body.TargetUsername)
	if body.TargetUsername == "" {
		http.Error(w, "target_username is required", http.StatusBadRequest)
		return
	}

	// Extract the raw token for later match creation.
	rawToken := r.Header.Get("Authorization")[7:]

	if err := h.cm.SendChallenge(claims.UserID, claims.Username, rawToken, body.TargetUsername); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "sent"})
}

// Pending handles GET /challenge/pending.
func (h *ChallengeHTTP) Pending(w http.ResponseWriter, r *http.Request) {
	claims, err := h.userFromRequest(r)
	if err != nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	challenges := h.cm.GetPending(claims.UserID)
	if challenges == nil {
		challenges = []model.ChallengeInfo(nil)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(challenges)
}

// Accept handles POST /challenge/accept.
// Request body: {"from_username": "..."}
func (h *ChallengeHTTP) Accept(w http.ResponseWriter, r *http.Request) {
	claims, err := h.userFromRequest(r)
	if err != nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	var body struct {
		FromUsername string `json:"from_username"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}
	body.FromUsername = strings.TrimSpace(body.FromUsername)
	if body.FromUsername == "" {
		http.Error(w, "from_username is required", http.StatusBadRequest)
		return
	}

	rawToken := r.Header.Get("Authorization")[7:]
	info, err := h.cm.AcceptChallenge(claims.UserID, body.FromUsername, rawToken, h.mm)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(info)
}

// Reject handles POST /challenge/reject.
// Request body: {"from_username": "..."}
func (h *ChallengeHTTP) Reject(w http.ResponseWriter, r *http.Request) {
	claims, err := h.userFromRequest(r)
	if err != nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	var body struct {
		FromUsername string `json:"from_username"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}
	body.FromUsername = strings.TrimSpace(body.FromUsername)
	if body.FromUsername == "" {
		http.Error(w, "from_username is required", http.StatusBadRequest)
		return
	}

	if err := h.cm.RejectChallenge(claims.UserID, body.FromUsername); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "rejected"})
}
