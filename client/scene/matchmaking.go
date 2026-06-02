//go:build js && wasm

package scene

import (
	"github.com/hajimehoshi/ebiten/v2"

	"connect4/client/assets"
)

// Matchmaking shows a searching animation and waits for a match.
// The player can cancel at any time before being paired.
type Matchmaking struct {
	mgr  *Manager
	bg   *ebiten.Image
}

// NewMatchmaking creates the Matchmaking scene.
func NewMatchmaking(mgr *Manager) *Matchmaking {
	return &Matchmaking{
		mgr: mgr,
		bg:  assets.MustLoadImage("images/backgrounds/matchmaking.png"),
	}
}

func (s *Matchmaking) Update() error {
	// TODO:
	// - On entry: open WSConn, send auth message, then send "queue" message.
	// - Poll recv channel each tick.
	// - On "matched" message: store WSConn in a shared place, mgr.Navigate(IDGame).
	// - On cancel button click: send "cancel" over WS, mgr.Back().
	return nil
}

func (s *Matchmaking) Draw(screen *ebiten.Image) {
	// TODO: draw bg, searching animation, cancel button.
	opts := &ebiten.DrawImageOptions{}
	screen.DrawImage(s.bg, opts)
}
