//go:build js && wasm

package scene

import (
	"github.com/hajimehoshi/ebiten/v2"

	"connect4/client/assets"
)

// Leaderboard shows all players ranked by ELO descending.
type Leaderboard struct {
	mgr     *Manager
	bg      *ebiten.Image
	loaded  bool
	loading bool
	errMsg  string

	entries []leaderboardEntry
	scroll  int
}

type leaderboardEntry struct {
	rank     int
	username string
	elo      int
	wins     int
	losses   int
	draws    int
}

// NewLeaderboard creates the Leaderboard scene.
func NewLeaderboard(mgr *Manager) *Leaderboard {
	return &Leaderboard{
		mgr: mgr,
		bg:  assets.MustLoadImage("images/backgrounds/leaderboard.png"),
	}
}

func (s *Leaderboard) Update() error {
	// TODO:
	// - On first update: fire GET /leaderboard asynchronously. Populate entries.
	// - Scroll with mouse wheel.
	// - On username click: mgr.Navigate(IDProfileOther) with that username.
	// - Back button: mgr.Back().
	return nil
}

func (s *Leaderboard) Draw(screen *ebiten.Image) {
	// TODO:
	// - Draw bg.
	// - Draw table header: Rank | Player | ELO | W | L | D
	// - Draw each entry row (usernames clickable).
	// - Highlight the logged-in player's own row.
	// - Draw back button.
	opts := &ebiten.DrawImageOptions{}
	screen.DrawImage(s.bg, opts)
}
