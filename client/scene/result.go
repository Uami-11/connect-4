//go:build js && wasm

package scene

import (
	"fmt"
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text"
	"golang.org/x/image/font/basicfont"

	"connect4/client/assets"
	"connect4/client/session"
	"connect4/client/ui"
)

type Result struct {
	mgr       *Manager
	bg        *ebiten.Image
	outcome   string
	winColor  string
	eloBefore int
	eloAfter  int
	eloDelta  int
	menuBtn   *ui.Button
	overlay   *ebiten.Image
}

func NewResult(mgr *Manager, outcome, winColor string, eloBefore, eloAfter, eloDelta int) *Result {
	btnW, btnH := 220, 44
	btnX := (1024 - btnW) / 2
	btnY := 600

	s := &Result{
		mgr:       mgr,
		bg:        assets.MustLoadImage("images/backgrounds/game.png"),
		outcome:   outcome,
		winColor:  winColor,
		eloBefore: eloBefore,
		eloAfter:  eloAfter,
		eloDelta:  eloDelta,
		overlay:   ebiten.NewImage(1024, 768),
	}
	s.overlay.Fill(color.RGBA{0, 0, 0, 0x88})

	s.menuBtn = ui.NewButton(btnX, btnY, btnW, btnH, "Back to Menu", func() {
		session.CurrentWS = nil
		session.CurrentMatchColor = 0
		session.CurrentMatchOpponent = ""
		session.CurrentResult = nil
		mgr.Navigate(IDMenu)
	})
	s.menuBtn.BgColor = deepWalnut
	s.menuBtn.HoverColor = darkCyan
	s.menuBtn.TextColor = white

	return s
}

func (s *Result) Update() error {
	s.menuBtn.Update()
	return nil
}

func (s *Result) Draw(screen *ebiten.Image) {
	screen.DrawImage(s.bg, nil)
	screen.DrawImage(s.overlay, nil)

	var outcomeText string
	var outcomeColor color.Color
	switch s.outcome {
	case "win":
		outcomeText = "You Win!"
		outcomeColor = color.RGBA{0x40, 0xff, 0x40, 0xff}
	case "loss":
		outcomeText = "You Lose"
		outcomeColor = color.RGBA{0xff, 0x60, 0x60, 0xff}
	default:
		outcomeText = "Draw"
		outcomeColor = color.RGBA{0xff, 0xff, 0x40, 0xff}
	}
	b := text.BoundString(basicfont.Face7x13, outcomeText)
	tx := (1024 - b.Dx()) / 2
	text.Draw(screen, outcomeText, basicfont.Face7x13, tx, 250, outcomeColor)

	if s.outcome != "draw" {
		colorText := s.winColor + " wins"
		cb := text.BoundString(basicfont.Face7x13, colorText)
		ctx := (1024 - cb.Dx()) / 2
		text.Draw(screen, colorText, basicfont.Face7x13, ctx, 280, frostedMint)
	}

	delta := s.eloAfter - s.eloBefore
	var eloText string
	var eloColor color.Color
	if delta >= 0 {
		eloText = fmt.Sprintf("%d → %d (+%d)", s.eloBefore, s.eloAfter, delta)
		eloColor = color.RGBA{0x40, 0xff, 0x40, 0xff}
	} else {
		eloText = fmt.Sprintf("%d → %d (%d)", s.eloBefore, s.eloAfter, delta)
		eloColor = color.RGBA{0xff, 0x60, 0x60, 0xff}
	}
	eb := text.BoundString(basicfont.Face7x13, eloText)
	etx := (1024 - eb.Dx()) / 2
	text.Draw(screen, eloText, basicfont.Face7x13, etx, 350, eloColor)

	s.menuBtn.Draw(screen)
}
