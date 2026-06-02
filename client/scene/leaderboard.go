//go:build js && wasm

package scene

import (
	"image/color"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hajimehoshi/ebiten/v2/text"
	"golang.org/x/image/font/basicfont"

	"connect4/client/assets"
	"connect4/client/net"
	"connect4/client/session"
	"connect4/client/ui"
)

type leaderboardEntry struct {
	rank     int
	username string
	elo      int
	wins     int
	losses   int
	draws    int
}

// Leaderboard shows all players ranked by ELO descending.
type Leaderboard struct {
	mgr     *Manager
	bg      *ebiten.Image
	loaded  bool
	loading bool
	errMsg  string

	req     *net.PendingRequest
	entries []leaderboardEntry
	scrollY int
	backBtn *ui.Button

	fetchStart time.Time
}

// NewLeaderboard creates the Leaderboard scene.
func NewLeaderboard(mgr *Manager) *Leaderboard {
	s := &Leaderboard{
		mgr:        mgr,
		bg:         assets.MustLoadImage("images/backgrounds/leaderboard.png"),
		fetchStart: time.Now(),
	}

	btnW, btnH := 120, 36
	s.backBtn = ui.NewButton(60, 700, btnW, btnH, "Back", func() {
		mgr.Back()
	})
	s.backBtn.BgColor = deepWalnut
	s.backBtn.HoverColor = darkCyan
	s.backBtn.TextColor = white

	return s
}

func (s *Leaderboard) Update() error {
	// Start request on first update.
	if !s.loaded && !s.loading && s.req == nil && s.errMsg == "" {
		s.req = net.StartGet("/leaderboard", session.Current.Token)
		s.loading = true
	}

	// Poll request.
	if s.req != nil {
		net.Poll(s.req)
		if s.req.Done {
			req := s.req
			s.req = nil
			s.loading = false
			var resp []struct {
				Rank     int    `json:"rank"`
				Username string `json:"username"`
				ELO      int    `json:"elo"`
				Wins     int    `json:"wins"`
				Losses   int    `json:"losses"`
				Draws    int    `json:"draws"`
			}
			if err := net.DecodeResult(req, &resp); err != nil {
				s.errMsg = err.Error()
			} else {
				for _, e := range resp {
					s.entries = append(s.entries, leaderboardEntry{
						rank:     e.Rank,
						username: e.Username,
						elo:      e.ELO,
						wins:     e.Wins,
						losses:   e.Losses,
						draws:    e.Draws,
					})
				}
				s.loaded = true
			}
		}
	}

	// Back button.
	s.backBtn.Update()

	// Scrolling and click handling.
	if s.loaded && len(s.entries) > 0 {
		_, dy := ebiten.Wheel()
		if dy != 0 {
			rowH := 28
			visibleH := 340
			maxScroll := len(s.entries)*rowH - visibleH
			if maxScroll < 0 {
				maxScroll = 0
			}
			s.scrollY -= int(dy) * 30
			if s.scrollY < 0 {
				s.scrollY = 0
			}
			if s.scrollY > maxScroll {
				s.scrollY = maxScroll
			}
		}

		if inpututil.IsMouseButtonJustReleased(ebiten.MouseButtonLeft) {
			mx, my := ebiten.CursorPosition()
			rowH := 28
			startY := 120
			for i, e := range s.entries {
				ry := startY + i*rowH - s.scrollY
				if ry < 100 || ry > 480 {
					continue
				}
				if my >= ry && my < ry+rowH && mx >= 160 && mx < 380 {
					session.CurrentOtherUsername = e.username
					s.mgr.Navigate(IDProfileOther)
					return nil
				}
			}
		}
	}

	return nil
}

func (s *Leaderboard) Draw(screen *ebiten.Image) {
	screen.DrawImage(s.bg, nil)

	// Semi-transparent overlay.
	boxW, boxH := 904, 660
	boxX := (1024 - boxW) / 2
	boxY := 54
	boxImg := ebiten.NewImage(boxW, boxH)
	boxImg.Fill(color.RGBA{0x0a, 0x0a, 0x1a, 0xaa})
	opts := &ebiten.DrawImageOptions{}
	opts.GeoM.Translate(float64(boxX), float64(boxY))
	screen.DrawImage(boxImg, opts)

	if s.loading {
		lt := "loading..."
		lb := text.BoundString(basicfont.Face7x13, lt)
		text.Draw(screen, lt, basicfont.Face7x13, (1024-lb.Dx())/2, 400, lightGray)
		return
	}

	if s.errMsg != "" {
		et := "Error: " + s.errMsg
		eb := text.BoundString(basicfont.Face7x13, et)
		text.Draw(screen, et, basicfont.Face7x13, (1024-eb.Dx())/2, 400, matchErrColor)
		s.backBtn.Draw(screen)
		return
	}

	if !s.loaded {
		return
	}

	// Title.
	title := "Leaderboard"
	tb := text.BoundString(basicfont.Face7x13, title)
	text.Draw(screen, title, basicfont.Face7x13, (1024-tb.Dx())/2, boxY+30, frostedMint)

	lx := 80

	// Header row.
	headerY := boxY + 55
	text.Draw(screen, "#", basicfont.Face7x13, lx, headerY, lightGray)
	text.Draw(screen, "Player", basicfont.Face7x13, 160, headerY, lightGray)
	text.Draw(screen, "ELO", basicfont.Face7x13, 400, headerY, lightGray)
	text.Draw(screen, "W", basicfont.Face7x13, 500, headerY, lightGray)
	text.Draw(screen, "L", basicfont.Face7x13, 540, headerY, lightGray)
	text.Draw(screen, "D", basicfont.Face7x13, 580, headerY, lightGray)

	// Separator line.
	sep := ebiten.NewImage(boxW-20, 1)
	sep.Fill(lightGray)
	sopts := &ebiten.DrawImageOptions{}
	sopts.GeoM.Translate(float64(boxX+10), float64(headerY+4))
	screen.DrawImage(sep, sopts)

	// Entry rows.
	rowH := 28
	startY := headerY + 10
	clipTop := boxY + 60
	clipBottom := boxY + boxH - 10

	for i, e := range s.entries {
		ry := startY + i*rowH - s.scrollY

		if ry+rowH < clipTop || ry > clipBottom {
			continue
		}

		// Determine colors.
		rowClr := frostedMint
		rowBg := (color.Color)(nil)
		if e.username == session.Current.Username {
			rowBg = color.RGBA{0x11, 0x9d, 0xa4, 0x33}
		}

		// Highlight own row.
		if rowBg != nil {
			hl := ebiten.NewImage(boxW-20, rowH)
			hl.Fill(rowBg)
			hopts := &ebiten.DrawImageOptions{}
			hopts.GeoM.Translate(float64(boxX+10), float64(ry))
			screen.DrawImage(hl, hopts)
		}

		// Rank.
		rankStr := itoa(e.rank)
		text.Draw(screen, rankStr, basicfont.Face7x13, lx, ry+rowH-6, rowClr)

		// Username (clickable).
		unClr := rowClr
		mx, my := ebiten.CursorPosition()
		if my >= ry && my < ry+rowH && mx >= 160 && mx < 380 {
			unClr = darkCyan
		}
		text.Draw(screen, e.username, basicfont.Face7x13, 160, ry+rowH-6, unClr)

		// ELO.
		eloStr := itoa(e.elo)
		text.Draw(screen, eloStr, basicfont.Face7x13, 400, ry+rowH-6, rowClr)

		// W/L/D.
		text.Draw(screen, itoa(e.wins), basicfont.Face7x13, 500, ry+rowH-6, profileGreen)
		text.Draw(screen, itoa(e.losses), basicfont.Face7x13, 540, ry+rowH-6, profileRed)
		text.Draw(screen, itoa(e.draws), basicfont.Face7x13, 580, ry+rowH-6, profileYellow)
	}

	// Back button.
	s.backBtn.Draw(screen)
}
