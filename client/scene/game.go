//go:build js && wasm

package scene

import (
	"github.com/hajimehoshi/ebiten/v2"

	"connect4/client/assets"
)

// Game is the live Connect 4 board scene.
type Game struct {
	mgr         *Manager
	bg          *ebiten.Image
	board       *ebiten.Image
	redToken    *ebiten.Image
	yellowToken *ebiten.Image

	// Board state received from server.
	cells       [6][7]int
	myColor     int    // 1 = red, 2 = yellow
	myTurn      bool
	opponentName string

	// Disconnect grace period.
	showDisconnect bool
	disconnectSecs int

	// Result — when set, transition to IDResult.
	result *gameResult
}

type gameResult struct {
	outcome   string // "win", "loss", "draw"
	winColor  string
	eloBefore int
	eloAfter  int
	eloDelta  int
}

// NewGame creates the Game scene.
func NewGame(mgr *Manager) *Game {
	return &Game{
		mgr:         mgr,
		bg:          assets.MustLoadImage("images/backgrounds/game.png"),
		board:       assets.MustLoadImage("images/main/board.png"),
		redToken:    assets.MustLoadImage("images/main/red_token.png"),
		yellowToken: assets.MustLoadImage("images/main/yellow_token.png"),
	}
}

func (s *Game) Update() error {
	// TODO:
	// - Each tick: drain WSConn recv channel.
	//   "state"               → update s.cells and s.myTurn
	//   "opponent_disconnected" → show s.showDisconnect = true; update timer
	//   "opponent_reconnected"  → hide disconnect overlay
	//   "result"              → store s.result; mgr.Navigate(IDResult)
	// - On mouse click: detect column from cursor X position.
	//   If s.myTurn: send "place" message over WS.
	return nil
}

func (s *Game) Draw(screen *ebiten.Image) {
	// TODO:
	// - Draw bg, then board.png overlay.
	// - For each cell in s.cells draw red_token or yellow_token at the right position.
	// - Highlight the column the cursor is hovering over.
	// - If s.showDisconnect: draw overlay with countdown timer.
	opts := &ebiten.DrawImageOptions{}
	screen.DrawImage(s.bg, opts)
	screen.DrawImage(s.board, opts)
}
