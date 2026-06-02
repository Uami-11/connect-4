//go:build js && wasm

package scene

import (
	"github.com/hajimehoshi/ebiten/v2"

	"connect4/client/assets"
)

// Profile shows the logged-in player's own profile and match history.
type Profile struct {
	mgr      *Manager
	bg       *ebiten.Image
	loaded   bool
	loading  bool
	errMsg   string

	username string
	elo      int
	wins     int
	losses   int
	draws    int
	history  []historyEntry
	scroll   int
}

type historyEntry struct {
	opponentName string
	outcome      string // "win", "loss", "draw"
	eloBefore    int
	eloAfter     int
	eloDelta     int
}

// NewProfile creates the own-profile scene.
func NewProfile(mgr *Manager) *Profile {
	return &Profile{
		mgr: mgr,
		bg:  assets.MustLoadImage("images/backgrounds/user_info.png"),
	}
}

func (s *Profile) Update() error {
	// TODO:
	// - On first update: fire GET /profile/{session.Current.Username} asynchronously.
	//   Populate fields on response.
	// - Scroll through match history with mouse wheel.
	// - On opponent name click: mgr.Navigate(IDProfileOther) passing the username.
	// - On back button: mgr.Back().
	return nil
}

func (s *Profile) Draw(screen *ebiten.Image) {
	// TODO:
	// - Draw bg.
	// - Show username, ELO, wins/losses/draws summary.
	// - Draw scrollable match history list.
	//   Each row: opponent name (clickable), W/L/D, ELO delta (green/red).
	// - Draw back button.
	opts := &ebiten.DrawImageOptions{}
	screen.DrawImage(s.bg, opts)
}
