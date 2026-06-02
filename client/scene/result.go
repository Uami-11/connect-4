//go:build js && wasm

package scene

import (
	"github.com/hajimehoshi/ebiten/v2"

	"connect4/client/assets"
)

// Result shows the end-of-game outcome, winning color, and ELO change.
type Result struct {
	mgr      *Manager
	bg       *ebiten.Image
	outcome  string // "win", "loss", "draw"
	winColor string // "red", "yellow", "draw"
	eloBefore int
	eloAfter  int
	eloDelta  int
}

// NewResult creates the Result scene with the given outcome data.
func NewResult(mgr *Manager, outcome, winColor string, eloBefore, eloAfter, eloDelta int) *Result {
	return &Result{
		mgr:       mgr,
		bg:        assets.MustLoadImage("images/backgrounds/game.png"),
		outcome:   outcome,
		winColor:  winColor,
		eloBefore: eloBefore,
		eloAfter:  eloAfter,
		eloDelta:  eloDelta,
	}
}

func (s *Result) Update() error {
	// TODO: on "Back to Menu" button click → mgr.Navigate(IDMenu).
	return nil
}

func (s *Result) Draw(screen *ebiten.Image) {
	// TODO:
	// - Draw bg.
	// - Show large outcome text ("You Win!", "You Lose", "Draw").
	// - Show which color won (red / yellow token icon).
	// - Show ELO change: e.g. "1024 → 1042 (+18)".
	// - Draw "Back to Menu" button.
	opts := &ebiten.DrawImageOptions{}
	screen.DrawImage(s.bg, opts)
}
