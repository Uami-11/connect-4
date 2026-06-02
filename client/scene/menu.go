//go:build js && wasm

package scene

import (
	"github.com/hajimehoshi/ebiten/v2"

	"connect4/client/assets"
)

// Menu is the main menu with Find Match, Profile, Leaderboard, Sign Out.
type Menu struct {
	mgr *Manager
	bg  *ebiten.Image
}

// NewMenu creates the Menu scene.
func NewMenu(mgr *Manager) *Menu {
	return &Menu{
		mgr: mgr,
		bg:  assets.MustLoadImage("images/backgrounds/main_menu.png"),
	}
}

func (s *Menu) Update() error {
	// TODO: handle clicks on the four menu options.
	// Find Match  → mgr.Navigate(IDMatchmaking)
	// Profile     → mgr.Navigate(IDProfile)
	// Leaderboard → mgr.Navigate(IDLeaderboard)
	// Sign Out    → session.Current.Clear(); mgr.Reset()
	return nil
}

func (s *Menu) Draw(screen *ebiten.Image) {
	// TODO: draw bg, four menu option buttons.
	opts := &ebiten.DrawImageOptions{}
	screen.DrawImage(s.bg, opts)
}
