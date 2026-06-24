package game

import (
	"context"
	"sync"

	"connect4/server/internal/db"
)

// Matchmaker manages the waiting queue and all live matches.
type Matchmaker struct {
	mu      sync.Mutex
	waiting *Client            // at most one player waiting at a time
	matches map[string]*Match  // matchID → Match
	queries *db.Queries
}

// NewMatchmaker creates a ready Matchmaker.
func NewMatchmaker(queries *db.Queries) *Matchmaker {
	return &Matchmaker{
		matches: make(map[string]*Match),
		queries: queries,
	}
}

// Enqueue adds a client to the queue.
// If another player is already waiting, they are immediately paired and a
// Match is created. Returns the new Match, or nil if still waiting.
func (mm *Matchmaker) Enqueue(c *Client) *Match {
	mm.mu.Lock()
	defer mm.mu.Unlock()

	if mm.waiting == nil {
		mm.waiting = c
		return nil
	}

	// Pair the two players.
	p1 := mm.waiting
	mm.waiting = nil

	id := matchID(p1, c)
	m := NewMatch(id, p1, c, mm.queries, mm)
	mm.matches[id] = m
	return m
}

// Dequeue removes a waiting client from the queue (they cancelled).
// Returns true if they were actually in the queue.
func (mm *Matchmaker) Dequeue(c *Client) bool {
	mm.mu.Lock()
	defer mm.mu.Unlock()
	if mm.waiting == c {
		mm.waiting = nil
		return true
	}
	return false
}

// FindMatch returns the active match containing the given client token,
// used for reconnect attempts.
func (mm *Matchmaker) FindMatch(token string) *Match {
	mm.mu.Lock()
	defer mm.mu.Unlock()
	for _, m := range mm.matches {
		if m.p1.Token == token || m.p2.Token == token {
			return m
		}
	}
	return nil
}

// Remove deletes a finished match from the map.
func (mm *Matchmaker) Remove(id string) {
	mm.mu.Lock()
	defer mm.mu.Unlock()
	delete(mm.matches, id)
}

// UpdateLastActive sets the player's last_active timestamp to now.
func (mm *Matchmaker) UpdateLastActive(ctx context.Context, userID int) {
	mm.queries.UpdateLastActive(ctx, userID)
}

// GetELO returns the current ELO rating for a user.
func (mm *Matchmaker) GetELO(ctx context.Context, userID int) (int, error) {
	return mm.queries.GetELOByID(ctx, userID)
}

// matchID generates a stable key from two clients.
func matchID(p1, p2 *Client) string {
	return p1.Token + ":" + p2.Token
}
