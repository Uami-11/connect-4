package handler

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/gorilla/websocket"

	"connect4/server/internal/auth"
	"connect4/server/internal/game"
	"connect4/server/internal/model"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

// WS handles the /ws WebSocket endpoint.
type WS struct {
	mm        *game.Matchmaker
	jwtSecret string
}

// NewWS creates a WS handler.
func NewWS(mm *game.Matchmaker, jwtSecret string) *WS {
	return &WS{mm: mm, jwtSecret: jwtSecret}
}

// ServeHTTP upgrades the connection, authenticates the player via their
// first message, and drives the matchmaking + game loop.
func (h *WS) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println("ws upgrade:", err)
		return
	}

	// First message must be an auth message with the JWT.
	var authMsg struct {
		Type  string `json:"type"`
		Token string `json:"token"`
	}
	if err := conn.ReadJSON(&authMsg); err != nil || authMsg.Type != "auth" {
		conn.WriteJSON(model.WSMessage{Type: "error", Payload: map[string]string{"message": "first message must be auth"}})
		conn.Close()
		return
	}

	claims, err := auth.ParseToken(authMsg.Token, h.jwtSecret)
	if err != nil {
		conn.WriteJSON(model.WSMessage{Type: "error", Payload: map[string]string{"message": "invalid token"}})
		conn.Close()
		return
	}

	// Check if this is a reconnect attempt.
	if existing := h.mm.FindMatch(authMsg.Token); existing != nil {
		newClient := &game.Client{
			UserID:   claims.UserID,
			Username: claims.Username,
			Send:     make(chan []byte, 64),
			Token:    authMsg.Token,
		}
		if existing.TryRejoin(authMsg.Token, newClient) {
			log.Printf("player %s rejoined match\n", claims.Username)
			go writePump(conn, newClient)
			readPump(conn, newClient, existing, h.mm)
			return
		}
	}

	// New connection — create client and enqueue.
	client := &game.Client{
		UserID:   claims.UserID,
		Username: claims.Username,
		Send:     make(chan []byte, 64),
		Token:    authMsg.Token,
	}

	// Second message must be "queue" or "cancel".
	var queueMsg model.WSMessage
	if err := conn.ReadJSON(&queueMsg); err != nil {
		conn.Close()
		return
	}

	switch queueMsg.Type {
	case "queue":
		conn.WriteJSON(model.WSMessage{Type: "waiting"})
		match := h.mm.Enqueue(client)
		if match == nil {
			// Still waiting — just start the write pump and block on reads
			// until a match is created or they cancel.
			go writePump(conn, client)
			waitInQueue(conn, client, h.mm)
			return
		}
		go writePump(conn, client)
		readPump(conn, client, match, h.mm)
	default:
		conn.Close()
	}
}

// waitInQueue reads messages while the client waits for an opponent.
// It handles "cancel" to leave the queue, and after a match is created it
// forwards game messages (place, disconnect) to the match.
func waitInQueue(conn *websocket.Conn, c *game.Client, mm *game.Matchmaker) {
	defer func() {
		mm.Dequeue(c)
		// If a match exists, notify it of the disconnect.
		if m := mm.FindMatch(c.Token); m != nil {
			m.HandleDisconnect(c)
		}
		conn.Close()
		close(c.Send)
	}()
	for {
		_, raw, err := conn.ReadMessage()
		if err != nil {
			return
		}
		var msg model.WSMessage
		if err := json.Unmarshal(raw, &msg); err != nil {
			continue
		}
		switch msg.Type {
		case "cancel":
			mm.Dequeue(c)
			conn.WriteJSON(model.WSMessage{Type: "cancelled"})
			return
		case "place":
			// Match may have been created while we were waiting.
			if m := mm.FindMatch(c.Token); m != nil {
				m.HandleMessage(c, raw)
			}
		}
	}
}

// readPump reads messages from the WebSocket and forwards them to the match.
func readPump(conn *websocket.Conn, c *game.Client, m *game.Match, mm *game.Matchmaker) {
	defer func() {
		m.HandleDisconnect(c)
		conn.Close()
	}()
	for {
		_, raw, err := conn.ReadMessage()
		if err != nil {
			return
		}
		var msg model.WSMessage
		if err := json.Unmarshal(raw, &msg); err != nil {
			continue
		}
		if msg.Type == "place" {
			m.HandleMessage(c, raw)
		}
	}
}

// writePump drains c.Send and writes each message to the WebSocket.
func writePump(conn *websocket.Conn, c *game.Client) {
	defer conn.Close()
	for msg := range c.Send {
		if err := conn.WriteMessage(websocket.TextMessage, msg); err != nil {
			return
		}
	}
}
