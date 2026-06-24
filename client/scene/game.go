//go:build js && wasm

package scene

import (
	"encoding/json"
	"fmt"
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hajimehoshi/ebiten/v2/text"
	"golang.org/x/image/font/basicfont"

	"connect4/client/assets"
	"connect4/client/net"
	"connect4/client/session"
)

const (
	scale             = 0.68
	boardScreenX      = (1024 - 1078*scale) / 2
	boardScreenY      = 50
	hoverTokenScreenY = 10
	colorAssignFrames = 120
	colStep           = 153
	rowStep           = 151
	gridLeft          = 24
	gridTop           = 67
)

type gamePhase int

const (
	phaseColorAssign gamePhase = iota
	phasePlaying
	phaseDisconnected
)

type Game struct {
	mgr *Manager
	bg  *ebiten.Image
	brd *ebiten.Image
	red *ebiten.Image
	yel *ebiten.Image

	ws     *net.WSConn
	phase  gamePhase
	frames int

	cells        [6][7]int
	myColor      int
	myTurn       bool
	opponentName string

	hoverCol       int
	keyboardActive bool
	lastMouseCol   int
	disconnectSec  int

	overlay *ebiten.Image
}

func NewGame(mgr *Manager) *Game {
	ws := session.CurrentWS
	s := &Game{
		mgr:          mgr,
		bg:           assets.MustLoadImage("images/backgrounds/game.png"),
		brd:          assets.MustLoadImage("images/main/board.png"),
		red:          assets.MustLoadImage("images/main/red_token.png"),
		yel:          assets.MustLoadImage("images/main/yellow_token.png"),
		ws:           ws,
		phase:        phaseColorAssign,
		frames:       colorAssignFrames,
		myColor:      session.CurrentMatchColor,
		opponentName: session.CurrentMatchOpponent,
		myTurn:       session.CurrentMatchColor == 1,
		hoverCol:      -1,
		lastMouseCol:  -1,
		overlay:       ebiten.NewImage(1024, 768),
	}
	s.overlay.Fill(color.RGBA{0, 0, 0, 0x88})
	return s
}

func (s *Game) Update() error {
	if s.ws != nil {
		select {
		case msg := <-s.ws.Recv():
			s.handleMessage(msg)
		default:
		}
	}

	switch s.phase {
	case phaseColorAssign:
		s.frames--
		if s.frames <= 0 {
			s.phase = phasePlaying
		}

	case phasePlaying:
		mx, _ := ebiten.CursorPosition()

		scaledLeft := boardScreenX + gridLeft*scale
		scaledColW := float64(colStep) * scale
		c := int((float64(mx) - scaledLeft) / scaledColW)
		mouseCol := -1
		if c >= 0 && c < 7 {
			mouseCol = c
		}

		if inpututil.IsKeyJustPressed(ebiten.KeyLeft) || inpututil.IsKeyJustPressed(ebiten.KeyA) {
			if s.hoverCol <= 0 {
				s.hoverCol = 6
			} else {
				s.hoverCol--
			}
			s.keyboardActive = true
		}
		if inpututil.IsKeyJustPressed(ebiten.KeyRight) || inpututil.IsKeyJustPressed(ebiten.KeyD) {
			if s.hoverCol >= 6 {
				s.hoverCol = 0
			} else {
				s.hoverCol++
			}
			s.keyboardActive = true
		}

		if mouseCol >= 0 {
			if mouseCol != s.lastMouseCol {
				s.lastMouseCol = mouseCol
				s.hoverCol = mouseCol
				s.keyboardActive = false
			}
		} else {
			s.lastMouseCol = -1
			if !s.keyboardActive {
				s.hoverCol = -1
			}
		}

		if s.myTurn && s.hoverCol >= 0 {
			if inpututil.IsMouseButtonJustReleased(ebiten.MouseButtonLeft) ||
				inpututil.IsKeyJustPressed(ebiten.KeyEnter) ||
				inpututil.IsKeyJustPressed(ebiten.KeyDown) ||
				inpututil.IsKeyJustPressed(ebiten.KeyS) {
				s.placeToken(s.hoverCol)
			}
		}
	}

	return nil
}

func (s *Game) placeToken(col int) {
	msg := map[string]any{
		"type": "place",
		"payload": map[string]int{
			"column": col,
		},
	}
	b, _ := json.Marshal(msg)
	s.ws.Send(string(b))
}

func (s *Game) handleMessage(raw string) {
	var h struct {
		Type    string          `json:"type"`
		Payload json.RawMessage `json:"payload"`
	}
	if json.Unmarshal([]byte(raw), &h) != nil {
		return
	}

	switch h.Type {
	case "state":
		var p struct {
			Board [6][7]int `json:"board"`
			Turn  int       `json:"turn"`
		}
		if json.Unmarshal(h.Payload, &p) != nil {
			return
		}
		s.cells = p.Board
		s.myTurn = p.Turn == s.myColor

	case "opponent_disconnected":
		var p struct {
			SecondsRemaining int `json:"seconds_remaining"`
		}
		if json.Unmarshal(h.Payload, &p) != nil {
			return
		}
		s.disconnectSec = p.SecondsRemaining
		s.phase = phaseDisconnected

	case "opponent_reconnected":
		s.phase = phasePlaying

	case "result":
		var p struct {
			Outcome   string `json:"outcome"`
			WinColor  string `json:"win_color"`
			ELOBefore int    `json:"elo_before"`
			ELOAfter  int    `json:"elo_after"`
			ELODelta  int    `json:"elo_delta"`
		}
		if json.Unmarshal(h.Payload, &p) != nil {
			return
		}
		session.CurrentResult = &session.GameResult{
			Outcome:   p.Outcome,
			WinColor:  p.WinColor,
			ELOBefore: p.ELOBefore,
			ELOAfter:  p.ELOAfter,
			ELODelta:  p.ELODelta,
		}
		s.mgr.Navigate(IDResult)
	}
}

func (s *Game) Draw(screen *ebiten.Image) {
	screen.DrawImage(s.bg, nil)

	opts := &ebiten.DrawImageOptions{}
	opts.GeoM.Scale(scale, scale)
	opts.GeoM.Translate(boardScreenX, boardScreenY)
	screen.DrawImage(s.brd, opts)

	for r := 0; r < 6; r++ {
		for c := 0; c < 7; c++ {
			player := s.cells[r][c]
			if player == 0 {
				continue
			}
			img := s.red
			if player == 2 {
				img = s.yel
			}
			imageRow := 5 - r
			tx := boardScreenX + (gridLeft+float64(c)*colStep)*scale
			ty := boardScreenY + (gridTop+float64(imageRow)*rowStep)*scale
			opts := &ebiten.DrawImageOptions{}
			opts.GeoM.Scale(scale, scale)
			opts.GeoM.Translate(tx, ty)
			screen.DrawImage(img, opts)
		}
	}

	if s.phase == phasePlaying && s.hoverCol >= 0 && s.myTurn {
		img := s.red
		if s.myColor == 2 {
			img = s.yel
		}
		hx := boardScreenX + (gridLeft+float64(s.hoverCol)*colStep)*scale
		opts := &ebiten.DrawImageOptions{}
		opts.GeoM.Scale(scale, scale)
		opts.GeoM.Translate(hx, hoverTokenScreenY)
		screen.DrawImage(img, opts)
	}

	if s.phase == phaseColorAssign {
		screen.DrawImage(s.overlay, nil)
		colorName := "Red"
		col := color.RGBA{0xff, 0x40, 0x40, 0xff}
		if s.myColor == 2 {
			colorName = "Yellow"
			col = color.RGBA{0xff, 0xff, 0x40, 0xff}
		}
		txt := "You are " + colorName
		b := text.BoundString(basicfont.Face7x13, txt)
		tx := (1024 - b.Dx()) / 2
		text.Draw(screen, txt, basicfont.Face7x13, tx, 768/2-10, col)

		sub := "vs " + s.opponentName
		sb := text.BoundString(basicfont.Face7x13, sub)
		stx := (1024 - sb.Dx()) / 2
		text.Draw(screen, sub, basicfont.Face7x13, stx, 768/2+14, frostedMint)
	}

	if s.phase == phaseDisconnected {
		screen.DrawImage(s.overlay, nil)
		txt := "Opponent disconnected — " + fmt.Sprintf("%ds", s.disconnectSec)
		b := text.BoundString(basicfont.Face7x13, txt)
		tx := (1024 - b.Dx()) / 2
		text.Draw(screen, txt, basicfont.Face7x13, tx, 768/2, color.RGBA{0xff, 0xcc, 0x00, 0xff})
	}
}
