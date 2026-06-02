package handler

import (
	"encoding/json"
	"net/http"
	"strings"

	"connect4/server/internal/auth"
	"connect4/server/internal/db"
	"connect4/server/internal/model"
)

// Auth handles POST /register and POST /login.
type Auth struct {
	queries   *db.Queries
	jwtSecret string
}

// NewAuth creates an Auth handler.
func NewAuth(queries *db.Queries, jwtSecret string) *Auth {
	return &Auth{queries: queries, jwtSecret: jwtSecret}
}

// Register handles POST /register.
func (a *Auth) Register(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}
	body.Username = strings.TrimSpace(body.Username)
	if len(body.Username) < 3 || len(body.Username) > 20 {
		http.Error(w, "username must be 3–20 characters", http.StatusBadRequest)
		return
	}
	if len(body.Password) < 6 {
		http.Error(w, "password must be at least 6 characters", http.StatusBadRequest)
		return
	}

	hash, err := auth.HashPassword(body.Password)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	id, err := a.queries.CreateUser(r.Context(), body.Username, hash)
	if err != nil {
		// Duplicate username gives a unique constraint error.
		if strings.Contains(err.Error(), "unique") || strings.Contains(err.Error(), "duplicate") {
			http.Error(w, "username already taken", http.StatusConflict)
			return
		}
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	token, err := auth.GenerateToken(id, body.Username, a.jwtSecret)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{
		"token":    token,
		"username": body.Username,
		"elo":      1000,
	})
}

// Login handles POST /login.
func (a *Auth) Login(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	user, err := a.queries.GetUserByUsername(r.Context(), body.Username)
	if err != nil {
		http.Error(w, "invalid username or password", http.StatusUnauthorized)
		return
	}
	if err := auth.CheckPassword(user.PasswordHash, body.Password); err != nil {
		http.Error(w, "invalid username or password", http.StatusUnauthorized)
		return
	}

	token, err := auth.GenerateToken(user.ID, user.Username, a.jwtSecret)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{
		"token":    token,
		"username": user.Username,
		"elo":      user.ELO,
	})
}

// Leaderboard handles GET /leaderboard.
type Leaderboard struct {
	queries *db.Queries
}

// NewLeaderboard creates a Leaderboard handler.
func NewLeaderboard(queries *db.Queries) *Leaderboard {
	return &Leaderboard{queries: queries}
}

// Get handles GET /leaderboard.
func (l *Leaderboard) Get(w http.ResponseWriter, r *http.Request) {
	entries, err := l.queries.GetLeaderboard(r.Context())
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	if entries == nil {
		entries = []model.LeaderboardEntry{} // send [] not null
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(entries)
}

// Profile handles GET /profile/{username}.
type Profile struct {
	queries *db.Queries
}

// NewProfile creates a Profile handler.
func NewProfile(queries *db.Queries) *Profile {
	return &Profile{queries: queries}
}

// Get handles GET /profile/{username}.
func (p *Profile) Get(w http.ResponseWriter, r *http.Request) {
	// Go 1.22 ServeMux supports {username} path params.
	username := r.PathValue("username")
	if username == "" {
		http.Error(w, "missing username", http.StatusBadRequest)
		return
	}

	profile, err := p.queries.GetPublicProfile(r.Context(), username)
	if err != nil {
		if strings.Contains(err.Error(), "no rows") {
			http.Error(w, "user not found", http.StatusNotFound)
			return
		}
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(profile)
}
