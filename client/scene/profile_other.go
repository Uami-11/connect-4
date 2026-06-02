//go:build js && wasm

package scene

import (
	"github.com/hajimehoshi/ebiten/v2"

	"connect4/client/assets"
)

// ProfileOther shows another player's public profile.
// It is the same layout as Profile but for any username, read-only.
type ProfileOther struct {
	mgr           *Manager
	bg            *ebiten.Image
	targetUsername string
	loaded        bool
	loading       bool
	errMsg        string

	elo     int
	wins    int
	losses  int
	draws   int
	history []historyEntry
	scroll  int
}

// NewProfileOther creates a public profile scene for the given username.
func NewProfileOther(mgr *Manager, username string) *ProfileOther {
	return &ProfileOther{
		mgr:           mgr,
		bg:            assets.MustLoadImage("images/backgrounds/user_info.png"),
		targetUsername: username,
	}
}

func (s *ProfileOther) Update() error {
	// TODO:
	// - On first update: fire GET /profile/{targetUsername} asynchronously.
	// - Opponent name clicks in history: mgr.Navigate(IDProfileOther) for that username.
	// - Back button: mgr.Back() (returns to wherever they came from).
	return nil
}

func (s *ProfileOther) Draw(screen *ebiten.Image) {
	// TODO: same layout as Profile but shows targetUsername's data.
	opts := &ebiten.DrawImageOptions{}
	screen.DrawImage(s.bg, opts)
}
