package game

import (
	"context"
	"encoding/json"
	"log"
	"sync"
	"time"

	"connect4/server/internal/db"
	"connect4/server/internal/elo"
	"connect4/server/internal/model"
)

const reconnectWindow = 30 * time.Second

// MatchState represents the lifecycle of a live match.
type MatchState int

const (
	StateActive       MatchState = iota
	StateReconnecting            // one player has disconnected; grace period running
	StateFinished
)

// Client is a connected WebSocket player inside a match.
type Client struct {
	UserID   int
	Username string
	ELO      int
	Color    int    // Red or Yellow
	Send     chan []byte
	Token    string // JWT used for reconnect identification
}

// Match holds the complete state of one live game.
type Match struct {
	mu      sync.Mutex
	id      string
	board   Board
	p1      *Client // Red, always moves first
	p2      *Client // Yellow
	turn    int     // Red or Yellow
	state   MatchState
	queries *db.Queries

	// Reconnect state
	disconnected *Client    // the client who left
	reconnectCh  chan *Client // filled when they come back
	cancelTimer  context.CancelFunc

	mm *Matchmaker // owning matchmaker, for self-removal on finish
}

// NewMatch creates and immediately starts a match between two clients.
func NewMatch(id string, p1, p2 *Client, queries *db.Queries, mm *Matchmaker) *Match {
	p1.Color = Red
	p2.Color = Yellow
	m := &Match{
		id:          id,
		p1:          p1,
		p2:          p2,
		turn:        Red,
		state:       StateActive,
		queries:     queries,
		reconnectCh: make(chan *Client, 1),
		mm:          mm,
	}
	m.sendMatched()
	go m.readLoop(p1, p2)
	go m.readLoop(p2, p1)
	return m
}

// sendMatched notifies both clients the match has started.
func (m *Match) sendMatched() {
	send := func(c *Client, opponentName string, yourTurn bool) {
		color := "red"
		if c.Color == Yellow {
			color = "yellow"
		}
		msg := model.WSMessage{
			Type: "matched",
			Payload: model.MatchedPayload{
				OpponentName: opponentName,
				YourColor:    color,
				YourTurn:     yourTurn,
			},
		}
		b, _ := json.Marshal(msg)
		c.Send <- b
	}
	send(m.p1, m.p2.Username, true)
	send(m.p2, m.p1.Username, false)
	m.broadcastState()
}

// readLoop reads messages from one client and processes them.
func (m *Match) readLoop(c *Client, opponent *Client) {
	// The WebSocket handler fills c.Send; this loop reads from c.
	// Actual ws.ReadMessage calls happen in handler/ws.go which
	// forwards decoded messages here via HandleMessage.
	_ = c
	_ = opponent
	// The handler calls HandleMessage and HandleDisconnect directly.
}

// HandleMessage processes a decoded "place" message from a client.
func (m *Match) HandleMessage(c *Client, raw []byte) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.state != StateActive {
		return
	}

	var msg model.WSMessage
	if err := json.Unmarshal(raw, &msg); err != nil {
		return
	}

	// Forfeit allowed regardless of turn.
	if msg.Type == "forfeit" {
		m.handleForfeit(c)
		return
	}

	if m.turn != c.Color {
		m.sendError(c, "not your turn")
		return
	}
	if msg.Type != "place" {
		return
	}

	var payload model.PlacePayload
	b, _ := json.Marshal(msg.Payload)
	if err := json.Unmarshal(b, &payload); err != nil {
		return
	}

	row, err := m.board.Drop(payload.Column, c.Color)
	if err != nil {
		m.sendError(c, err.Error())
		return
	}
	_ = row

	// Check win
	if m.board.CheckWin(c.Color) {
		m.broadcastState()
		m.finish(c, opponent(c, m.p1, m.p2))
		return
	}
	// Check draw
	if m.board.IsFull() {
		m.broadcastState()
		m.finish(nil, nil)
		return
	}

	// Advance turn
	if m.turn == Red {
		m.turn = Yellow
	} else {
		m.turn = Red
	}
	m.broadcastState()
}

// HandleDisconnect is called when a client's WebSocket connection drops.
func (m *Match) HandleDisconnect(c *Client) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.state != StateActive {
		return
	}
	m.state = StateReconnecting
	m.disconnected = c
	other := opponent(c, m.p1, m.p2)

	log.Printf("match %s: %s disconnected — grace period starting\n", m.id, c.Username)

	ctx, cancel := context.WithTimeout(context.Background(), reconnectWindow)
	m.cancelTimer = cancel

	// Notify the connected player with a countdown.
	go func() {
		ticker := time.NewTicker(time.Second)
		defer ticker.Stop()
		remaining := int(reconnectWindow.Seconds())
		for {
			select {
			case <-ctx.Done():
				// Timer expired — forfeit the disconnected player.
				m.mu.Lock()
				if m.state == StateReconnecting {
					log.Printf("match %s: %s forfeited (timeout)\n", m.id, c.Username)
					m.state = StateActive
					m.finish(other, c)
				}
				m.mu.Unlock()
				return
			case rejoined := <-m.reconnectCh:
				// Player came back.
				m.mu.Lock()
				m.state = StateActive
				m.disconnected = nil
				log.Printf("match %s: %s reconnected\n", m.id, rejoined.Username)
				notifyReconnected := model.WSMessage{Type: "opponent_reconnected"}
				b, _ := json.Marshal(notifyReconnected)
				other.Send <- b
				m.broadcastState()
				m.mu.Unlock()
				return
			case <-ticker.C:
				remaining--
				msg := model.WSMessage{
					Type:    "opponent_disconnected",
					Payload: model.DisconnectPayload{SecondsRemaining: remaining},
				}
				b, _ := json.Marshal(msg)
				other.Send <- b
			}
		}
	}()
}

// TryRejoin checks if the given token belongs to the disconnected player
// and reconnects them if so.
// drainChan transfers any buffered messages from src to dst without blocking.
func drainChan(src, dst chan []byte) {
	for {
		select {
		case msg := <-src:
			select {
			case dst <- msg:
			default:
			}
		default:
			return
		}
	}
}

func (m *Match) TryRejoin(token string, newClient *Client) bool {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Match by token to either player (supports both reconnect and challenge-accept join).
	if m.p1.Token == token {
		drainChan(m.p1.Send, newClient.Send)
		m.p1 = newClient
		if m.state == StateReconnecting && m.disconnected == m.p1 {
			m.reconnectCh <- newClient
		}
		return true
	}
	if m.p2.Token == token {
		drainChan(m.p2.Send, newClient.Send)
		m.p2 = newClient
		if m.state == StateReconnecting && m.disconnected == m.p2 {
			m.reconnectCh <- newClient
		}
			return true
	}
	return false
}

// InjectClient replaces a client in an active match by token (used for
// challenge-accepted joins where TryRejoin won't fire because the match
// is StateActive, not StateReconnecting). Preserves Color from the
// replaced client and drains any buffered messages.
func (m *Match) InjectClient(token string, newClient *Client) bool {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.state == StateFinished {
		return false
	}

	if m.p1.Token == token {
		drainChan(m.p1.Send, newClient.Send)
		newClient.Color = m.p1.Color
		m.p1 = newClient
		return true
	}
	if m.p2.Token == token {
		drainChan(m.p2.Send, newClient.Send)
		newClient.Color = m.p2.Color
		m.p2 = newClient
		return true
	}
	return false
}

// handleForfeit processes a player forfeiting the match.
func (m *Match) handleForfeit(c *Client) {
	if m.state != StateActive {
		m.sendError(c, "game not active")
		return
	}
	other := opponent(c, m.p1, m.p2)
	m.broadcastState()
	m.finish(other, c)
}

// finish ends the match, calculates ELO, and notifies both players.
// winner is nil for a draw; loser is nil for a draw.
func (m *Match) finish(winner, loser *Client) {
	m.state = StateFinished
	if m.cancelTimer != nil {
		m.cancelTimer()
	}

	var outcome float64
	if winner == nil {
		outcome = elo.Draw
	} else if winner == m.p1 {
		outcome = elo.Win
	} else {
		outcome = elo.Loss
	}

	new1, new2, delta := elo.Calculate(m.p1.ELO, m.p2.ELO, outcome)

	// Determine winner ID for DB.
	var winnerID *int
	if winner != nil {
		id := winner.UserID
		winnerID = &id
	}

	// Persist asynchronously — never block the game goroutine on DB.
	go func() {
		ctx := context.Background()
		_ = m.queries.UpdateELO(ctx, m.p1.UserID, new1)
		_ = m.queries.UpdateELO(ctx, m.p2.UserID, new2)
		_ = m.queries.CreateMatch(ctx, &model.Match{
			Player1ID:        m.p1.UserID,
			Player2ID:        m.p2.UserID,
			WinnerID:         winnerID,
			Player1ELOBefore: m.p1.ELO,
			Player2ELOBefore: m.p2.ELO,
			ELODelta:         delta,
		})
	}()

	// Notify clients.
	send := func(c *Client, newELO int) {
		outcome := "draw"
		if winner != nil {
			if c == winner {
				outcome = "win"
			} else {
				outcome = "loss"
			}
		}
		winColor := "draw"
		if winner != nil {
			if winner.Color == Red {
				winColor = "red"
			} else {
				winColor = "yellow"
			}
		}
		msg := model.WSMessage{
			Type: "result",
			Payload: model.ResultPayload{
				Outcome:   outcome,
				WinColor:  winColor,
				ELOBefore: c.ELO,
				ELOAfter:  newELO,
				ELODelta:  delta,
			},
		}
		b, _ := json.Marshal(msg)
		c.Send <- b
	}
	send(m.p1, new1)
	send(m.p2, new2)

	if m.mm != nil {
		m.mm.Remove(m.id)
	}
}

// broadcastState sends the current board and turn to both players.
func (m *Match) broadcastState() {
	msg := model.WSMessage{
		Type: "state",
		Payload: model.StatePayload{
			Board: m.board,
			Turn:  m.turn,
		},
	}
	b, _ := json.Marshal(msg)
	m.p1.Send <- b
	m.p2.Send <- b
}

func (m *Match) sendError(c *Client, text string) {
	msg := model.WSMessage{Type: "error", Payload: map[string]string{"message": text}}
	b, _ := json.Marshal(msg)
	c.Send <- b
}

// opponent returns the other player in the match.
func opponent(c, p1, p2 *Client) *Client {
	if c == p1 {
		return p2
	}
	return p1
}
