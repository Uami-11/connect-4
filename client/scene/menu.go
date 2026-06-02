//go:build js && wasm

package scene

import (
	"github.com/hajimehoshi/ebiten/v2"

	"connect4/client/assets"
	"connect4/client/session"
	"connect4/client/ui"
)

type Menu struct {
	mgr            *Manager
	bg             *ebiten.Image
	findMatchBtn   *ui.Button
	profileBtn     *ui.Button
	leaderboardBtn *ui.Button
	signOutBtn     *ui.Button
}

func NewMenu(mgr *Manager) *Menu {
	btnW, btnH := 260, 52
	btnX := (1024 - btnW) / 2

	// Vertical center: 4 buttons + 3 gaps
	totalH := 4*btnH + 3*12
	startY := (768 - totalH) / 2

	s := &Menu{
		mgr: mgr,
		bg:  assets.MustLoadImage("images/backgrounds/main_menu.png"),
	}

	s.findMatchBtn = ui.NewButton(btnX, startY, btnW, btnH, "Find Match", func() {
		mgr.Navigate(IDMatchmaking)
	})
	s.profileBtn = ui.NewButton(btnX, startY+btnH+12, btnW, btnH, "Profile", func() {
		mgr.Navigate(IDProfile)
	})
	s.leaderboardBtn = ui.NewButton(btnX, startY+(btnH+12)*2, btnW, btnH, "Leaderboard", func() {
		mgr.Navigate(IDLeaderboard)
	})
	s.signOutBtn = ui.NewButton(btnX, startY+(btnH+12)*3, btnW, btnH, "Sign Out", func() {
		session.Current.Clear()
		mgr.Reset()
	})

	for _, b := range []*ui.Button{s.findMatchBtn, s.profileBtn, s.leaderboardBtn, s.signOutBtn} {
		b.BgColor = deepWalnut
		b.TextColor = white
		b.HoverColor = darkCyan
	}
	s.signOutBtn.HoverColor = powderBlush

	return s
}

func (s *Menu) Update() error {
	s.findMatchBtn.Update()
	s.profileBtn.Update()
	s.leaderboardBtn.Update()
	s.signOutBtn.Update()
	return nil
}

func (s *Menu) Draw(screen *ebiten.Image) {
	screen.DrawImage(s.bg, nil)
	s.findMatchBtn.Draw(screen)
	s.profileBtn.Draw(screen)
	s.leaderboardBtn.Draw(screen)
	s.signOutBtn.Draw(screen)
}
