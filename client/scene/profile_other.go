//go:build js && wasm

package scene

import (
	"encoding/json"
	"image/color"
	"strings"
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

// ProfileOther shows another player's public profile.
// It is the same layout as Profile but for any username, read-only.
type ProfileOther struct {
	mgr            *Manager
	bg             *ebiten.Image
	targetUsername string
	loaded         bool
	loading        bool
	errMsg         string

	req *net.PendingRequest

	elo        int
	wins       int
	losses     int
	draws      int
	createdAt  string
	lastActive string
	online     bool
	history    []historyEntry

	scrollY      int
	backBtn      *ui.Button
	requestBtn   *ui.Button
	challengeReq *net.PendingRequest
	challengeErr string
}

// NewProfileOther creates a public profile scene for the given username.
func NewProfileOther(mgr *Manager, username string) *ProfileOther {
	s := &ProfileOther{
		mgr:            mgr,
		bg:             assets.MustLoadImage("images/backgrounds/user_info.png"),
		targetUsername: username,
	}

	btnW, btnH := 120, 36
	s.backBtn = ui.NewButton(60, 700, btnW, btnH, "Back", func() {
		mgr.Back()
	})
	s.backBtn.BgColor = deepWalnut
	s.backBtn.HoverColor = darkCyan
	s.backBtn.TextColor = white

	// Hide "Request Match" when viewing own profile.
	showRequest := username != session.Current.Username
	s.requestBtn = ui.NewButton(0, 0, 150, 36, "Request Match", func() {
		if s.challengeReq != nil {
			return
		}
		body := map[string]string{"target_username": s.targetUsername}
		s.challengeReq = net.StartPostAuth("/challenge/send", body, session.Current.Token)
	})
	s.requestBtn.BgColor = color.RGBA{0x44, 0xcc, 0x44, 0xff}
	s.requestBtn.HoverColor = color.RGBA{0x33, 0xaa, 0x33, 0xff}
	s.requestBtn.TextColor = white
	if !showRequest {
		s.requestBtn.SetHidden(true)
	}

	return s
}

func (s *ProfileOther) Update() error {
	// Start request on first update.
	if !s.loaded && !s.loading && s.req == nil && s.errMsg == "" {
		s.req = net.StartGet("/profile/"+s.targetUsername, session.Current.Token)
		s.loading = true
	}

	// Poll request.
	if s.req != nil {
		net.Poll(s.req)
		if s.req.Done {
			req := s.req
			s.req = nil
			s.loading = false
			var resp struct {
				Username     string            `json:"username"`
				ELO          int               `json:"elo"`
				Wins         int               `json:"wins"`
				Losses       int               `json:"losses"`
				Draws        int               `json:"draws"`
				CreatedAt    string            `json:"created_at"`
				LastActiveAt *string           `json:"last_active_at"`
				Online       bool              `json:"online"`
				History      []json.RawMessage `json:"history"`
			}
			if err := net.DecodeResult(req, &resp); err != nil {
				s.errMsg = err.Error()
			} else {
				s.populate(&resp)
				s.loaded = true
			}
		}
	}

	// Back button.
	s.backBtn.Update()

	// Poll challenge request response.
	if s.challengeReq != nil {
		net.Poll(s.challengeReq)
		if s.challengeReq.Done {
			req := s.challengeReq
			s.challengeReq = nil
			if req.Err() != nil {
				body := strings.ToLower(req.Err().Error())
				if strings.Contains(body, "already") || strings.Contains(body, "pending") {
					s.challengeErr = "already requested"
				} else {
					s.challengeErr = "failed to send request"
				}
			} else {
				s.challengeErr = ""
				s.requestBtn.SetDisabled(true)
				s.requestBtn.Text = "Requested"
			}
		}
	}

	// Position request button at far-right of inner box, aligned with username.
	if s.loaded {
		s.requestBtn.X = 794
		s.requestBtn.Y = 58
		if s.challengeReq == nil && !s.requestBtn.Disabled {
			s.requestBtn.Update()
		}
	}

	// Scrolling and click handling (only when loaded).
	if s.loaded && len(s.history) > 0 {
		_, dy := ebiten.Wheel()
		if dy != 0 {
			rowH := 28
			visibleH := 210
			maxScroll := len(s.history)*rowH - visibleH
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
			startY := 370
			for i, h := range s.history {
				ry := startY + i*rowH - s.scrollY
				if ry < 360 || ry > 570 {
					continue
				}
				if my >= ry && my < ry+rowH && mx >= 160 && mx < 400 {
					session.CurrentOtherUsername = h.opponentName
					s.mgr.Navigate(IDProfileOther)
					return nil
				}
			}
		}
	}

	return nil
}

func (s *ProfileOther) populate(resp *struct {
	Username     string            `json:"username"`
	ELO          int               `json:"elo"`
	Wins         int               `json:"wins"`
	Losses       int               `json:"losses"`
	Draws        int               `json:"draws"`
	CreatedAt    string            `json:"created_at"`
	LastActiveAt *string           `json:"last_active_at"`
	Online       bool              `json:"online"`
	History      []json.RawMessage `json:"history"`
}) {
	s.elo = resp.ELO
	s.wins = resp.Wins
	s.losses = resp.Losses
	s.draws = resp.Draws
	s.online = resp.Online

	if t, err := time.Parse(time.RFC3339, resp.CreatedAt); err == nil {
		s.createdAt = t.Format("2 Jan 2006")
	} else {
		s.createdAt = resp.CreatedAt
	}

	if resp.LastActiveAt != nil {
		if t, err := time.Parse(time.RFC3339, *resp.LastActiveAt); err == nil {
			s.lastActive = t.Format("2 Jan 2006 15:04")
		} else {
			s.lastActive = *resp.LastActiveAt
		}
	}

	s.history = nil
	for _, raw := range resp.History {
		var entry struct {
			OpponentName string `json:"opponent_name"`
			Outcome      string `json:"outcome"`
			ELOBefore    int    `json:"elo_before"`
			ELOAfter     int    `json:"elo_after"`
			ELODelta     int    `json:"elo_delta"`
			PlayedAt     string `json:"played_at"`
		}
		if err := json.Unmarshal(raw, &entry); err != nil {
			continue
		}
		playedAt := ""
		if t, err := time.Parse(time.RFC3339, entry.PlayedAt); err == nil {
			playedAt = t.Format("2 Jan")
		} else {
			playedAt = entry.PlayedAt
		}
		s.history = append(s.history, historyEntry{
			opponentName: entry.OpponentName,
			outcome:      entry.Outcome,
			eloBefore:    entry.ELOBefore,
			eloAfter:     entry.ELOAfter,
			eloDelta:     entry.ELODelta,
			playedAt:     playedAt,
		})
	}
}

func (s *ProfileOther) Draw(screen *ebiten.Image) {
	screen.DrawImage(s.bg, nil)

	// Semi-transparent overlay.
	boxW, boxH := 904, 728
	boxX := (1024 - boxW) / 2
	boxY := 20
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

	// --- Profile info ---
	lx := 80

	// Username (scaled 2x).
	s.drawScaledText(screen, s.targetUsername, float64(lx), 60, 2, frostedMint)

	// Request Match button (right of username).
	s.requestBtn.Draw(screen)
	if s.challengeErr != "" {
		text.Draw(screen, s.challengeErr, basicfont.Face7x13, s.requestBtn.X, s.requestBtn.Y+50, matchErrColor)
	}

	// ELO.
	eloStr := "Rating: " + itoa(s.elo)
	text.Draw(screen, eloStr, basicfont.Face7x13, lx, 130, frostedMint)

	// Joined.
	joinedStr := "Joined: " + s.createdAt
	text.Draw(screen, joinedStr, basicfont.Face7x13, lx, 170, lightGray)

	// Last active.
	if s.online {
		text.Draw(screen, "Status: Online now", basicfont.Face7x13, lx, 210, profileGreen)
	} else if s.lastActive != "" {
		activeStr := "Last seen: " + s.lastActive
		text.Draw(screen, activeStr, basicfont.Face7x13, lx, 210, lightGray)
	} else {
		text.Draw(screen, "Status: Offline", basicfont.Face7x13, lx, 210, lightGray)
	}

	// Stats.
	statsStr := "Wins: " + itoa(s.wins) + "  Losses: " + itoa(s.losses) + "  Draws: " + itoa(s.draws)
	text.Draw(screen, statsStr, basicfont.Face7x13, lx, 270, frostedMint)

	// --- Match history ---
	text.Draw(screen, "Match History", basicfont.Face7x13, lx, 340, frostedMint)

	rowH := 28
	startY := 370
	clipTop := 360
	clipBottom := 580

	for i, h := range s.history {
		ry := startY + i*rowH - s.scrollY

		if ry+rowH < clipTop || ry > clipBottom {
			continue
		}

		text.Draw(screen, h.playedAt, basicfont.Face7x13, lx, ry+rowH-6, lightGray)

		oppClr := powderBlush
		mx, my := ebiten.CursorPosition()
		if my >= ry && my < ry+rowH && mx >= 160 && mx < 400 {
			oppClr = darkCyan
		}
		text.Draw(screen, h.opponentName, basicfont.Face7x13, 160, ry+rowH-6, oppClr)

		outClr := profileGreen
		outLabel := "W"
		switch h.outcome {
		case "loss":
			outClr = profileRed
			outLabel = "L"
		case "draw":
			outClr = profileYellow
			outLabel = "D"
		}
		text.Draw(screen, outLabel, basicfont.Face7x13, 350, ry+rowH-6, outClr)

		deltaClr := profileGreen
		deltaSign := "+"
		if h.eloDelta < 0 {
			deltaClr = profileRed
			deltaSign = ""
		}
		deltaStr := deltaSign + itoa(h.eloDelta)
		text.Draw(screen, deltaStr, basicfont.Face7x13, 390, ry+rowH-6, deltaClr)
	}

	s.backBtn.Draw(screen)
}

func (s *ProfileOther) drawScaledText(screen *ebiten.Image, str string, x, y float64, scale float64, clr color.Color) {
	bounds := text.BoundString(basicfont.Face7x13, str)
	pad := 4
	w := bounds.Dx() + pad*2
	h := bounds.Dy() + pad*2
	if w < 1 {
		w = 1
	}
	if h < 1 {
		h = 1
	}
	img := ebiten.NewImage(w, h)
	text.Draw(img, str, basicfont.Face7x13, pad, pad+basicfont.Face7x13.Ascent, clr)
	opts := &ebiten.DrawImageOptions{}
	opts.GeoM.Scale(scale, scale)
	opts.GeoM.Translate(x, y)
	screen.DrawImage(img, opts)
}
