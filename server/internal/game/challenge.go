package game

import (
	"context"
	"fmt"
	"sync"
	"time"

	"connect4/server/internal/db"
	"connect4/server/internal/model"
)

// ChallengeStatus tracks the lifecycle of a player-to-player challenge.
type ChallengeStatus string

const (
	ChallengePending   ChallengeStatus = "pending"
	ChallengeAccepted  ChallengeStatus = "accepted"
	ChallengeRejected  ChallengeStatus = "rejected"
	ChallengeCancelled ChallengeStatus = "cancelled"
)

// Challenge represents one challenge request between two players.
type Challenge struct {
	FromUserID   int
	FromUsername string
	FromToken    string // JWT of the sender
	ToUserID     int
	ToUsername   string
	Status       ChallengeStatus
	CreatedAt    time.Time
}

// ChallengeManager stores pending challenges in memory.
type ChallengeManager struct {
	mu       sync.Mutex
	bySender map[int][]*Challenge   // sender userID -> challenges they sent
	byTarget map[int][]*Challenge   // target userID -> challenges they received
	queries  *db.Queries
}

// NewChallengeManager creates a ready ChallengeManager.
func NewChallengeManager(queries *db.Queries) *ChallengeManager {
	return &ChallengeManager{
		bySender: make(map[int][]*Challenge),
		byTarget: make(map[int][]*Challenge),
		queries:  queries,
	}
}

// SendChallenge creates a new challenge from one user to another.
// Returns an error if the target user doesn't exist or already has a pending challenge from this sender.
func (cm *ChallengeManager) SendChallenge(fromUserID int, fromUsername, fromToken, targetUsername string) error {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	// Look up target user.
	target, err := cm.queries.GetUserByUsername(context.Background(), targetUsername)
	if err != nil {
		return fmt.Errorf("target user not found")
	}
	if target.ID == fromUserID {
		return fmt.Errorf("cannot challenge yourself")
	}

	// Check for existing pending challenge from this sender to this target.
	for _, c := range cm.bySender[fromUserID] {
		if c.ToUserID == target.ID && c.Status == ChallengePending {
			return fmt.Errorf("challenge already pending")
		}
	}

	ch := &Challenge{
		FromUserID:   fromUserID,
		FromUsername: fromUsername,
		FromToken:    fromToken,
		ToUserID:     target.ID,
		ToUsername:   target.Username,
		Status:       ChallengePending,
		CreatedAt:    time.Now(),
	}
	cm.bySender[fromUserID] = append(cm.bySender[fromUserID], ch)
	cm.byTarget[target.ID] = append(cm.byTarget[target.ID], ch)
	return nil
}

// GetPending returns all challenges visible to a user (incoming pending + outgoing with status).
func (cm *ChallengeManager) GetPending(userID int) []model.ChallengeInfo {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	var result []model.ChallengeInfo

	// Incoming pending.
	for _, c := range cm.byTarget[userID] {
		if c.Status == ChallengePending || c.Status == ChallengeAccepted {
			info := model.ChallengeInfo{
				FromUsername: c.FromUsername,
				Status:       string(c.Status),
			}
			if c.Status == ChallengeAccepted {
				info.OpponentName = c.FromUsername
				info.YourColor = "red"
				info.YourTurn = true
			}
			result = append(result, info)
		}
	}

	// Outgoing.
	for _, c := range cm.bySender[userID] {
		info := model.ChallengeInfo{
			ToUsername: c.ToUsername,
			Status:     string(c.Status),
		}
		if c.Status == ChallengeAccepted {
			info.OpponentName = c.ToUsername
			info.YourColor = "red"
			info.YourTurn = true
		}
		result = append(result, info)
	}

	return result
}

// AcceptChallenge accepts a pending challenge. It creates a real Match and
// returns the challenge info including match details so the client can navigate to the game.
// acceptorToken is the JWT of the accepting player, used to identify them in the match.
func (cm *ChallengeManager) AcceptChallenge(acceptorID int, fromUsername, acceptorToken string, mm *Matchmaker) (*model.ChallengeInfo, error) {
	cm.mu.Lock()

	// Find the pending challenge.
	var ch *Challenge
	for _, c := range cm.byTarget[acceptorID] {
		if c.FromUsername == fromUsername && c.Status == ChallengePending {
			ch = c
			break
		}
	}
	if ch == nil {
		cm.mu.Unlock()
		return nil, fmt.Errorf("no pending challenge from %s", fromUsername)
	}

	ch.Status = ChallengeAccepted
	cm.mu.Unlock()

	// Get ELO for both players.
	ctx := context.Background()
	elo1, err1 := cm.queries.GetELOByID(ctx, ch.FromUserID)
	elo2, err2 := cm.queries.GetELOByID(ctx, ch.ToUserID)
	if err1 != nil || err2 != nil {
		return nil, fmt.Errorf("failed to get player ELO")
	}

	// Create the match using actual JWT tokens so TryRejoin can match them.
	// The clients aren't connected yet — placeholder Send channels are replaced
	// on actual WS connection via TryRejoin.
	p1 := &Client{
		UserID:   ch.FromUserID,
		Username: ch.FromUsername,
		ELO:      elo1,
		Token:    ch.FromToken,
		Send:     make(chan []byte, 64),
	}
	p2 := &Client{
		UserID:   ch.ToUserID,
		Username: ch.ToUsername,
		ELO:      elo2,
		Token:    acceptorToken,
		Send:     make(chan []byte, 64),
	}

	id := matchID(p1, p2)
	m := NewMatch(id, p1, p2, mm.queries, mm)
	mm.mu.Lock()
	mm.matches[id] = m
	mm.mu.Unlock()

	// Return info for the acceptor (they are p2).
	info := &model.ChallengeInfo{
		FromUsername: ch.FromUsername,
		ToUsername:   ch.ToUsername,
		Status:       string(ChallengeAccepted),
		OpponentName: ch.FromUsername,
		YourColor:    "yellow",
		YourTurn:     false,
	}
	return info, nil
}

// removeChallenge deletes a challenge from both lookup maps.
func (cm *ChallengeManager) removeChallenge(ch *Challenge) {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	senderList := cm.bySender[ch.FromUserID]
	for i, c := range senderList {
		if c == ch {
			cm.bySender[ch.FromUserID] = append(senderList[:i], senderList[i+1:]...)
			break
		}
	}

	targetList := cm.byTarget[ch.ToUserID]
	for i, c := range targetList {
		if c == ch {
			cm.byTarget[ch.ToUserID] = append(targetList[:i], targetList[i+1:]...)
			break
		}
	}
}

// RejectChallenge rejects a pending challenge.
func (cm *ChallengeManager) RejectChallenge(rejectorID int, fromUsername string) error {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	for _, c := range cm.byTarget[rejectorID] {
		if c.FromUsername == fromUsername && c.Status == ChallengePending {
			c.Status = ChallengeRejected
			return nil
		}
	}
	return fmt.Errorf("no pending challenge from %s", fromUsername)
}

// CancelChallenge cancels a challenge sent by the given user.
func (cm *ChallengeManager) CancelChallenge(senderID int, targetUsername string) error {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	for _, c := range cm.bySender[senderID] {
		if c.ToUsername == targetUsername && c.Status == ChallengePending {
			c.Status = ChallengeCancelled
			return nil
		}
	}
	return fmt.Errorf("no pending challenge to %s", targetUsername)
}


