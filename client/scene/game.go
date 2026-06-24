//go:build js && wasm

package scene

import (
	"encoding/json"
	"fmt"
	"image/color"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hajimehoshi/ebiten/v2/text"
	"golang.org/x/image/font/basicfont"

	"connect4/client/assets"
	"connect4/client/net"
	"connect4/client/session"
	"connect4/client/ui"
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
	phaseResult
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

	result       session.GameResult
	winCells     [][2]int
	ticks        int
	backBtn      *ui.Button
	forfeitBtn   *ui.Button
	forfeitSure  bool
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

	btnW, btnH := 220, 44
	btnX := (1024 - btnW) / 2
	s.backBtn = ui.NewButton(btnX, 600, btnW, btnH, "Back to Menu", func() {
		if s.ws != nil {
			s.ws.Close()
		}
		session.CurrentWS = nil
		session.CurrentMatchColor = 0
		session.CurrentMatchOpponent = ""
		session.CurrentResult = nil
		mgr.challengeHandledUser = ""
		mgr.Navigate(IDMenu)
	})
	s.backBtn.BgColor = deepWalnut
	s.backBtn.HoverColor = darkCyan
	s.backBtn.TextColor = white
	s.backBtn.SetHidden(true)

	s.forfeitBtn = ui.NewButton(20, 704, 140, 44, "Forfeit", func() {
		if !s.forfeitSure {
			s.forfeitSure = true
			s.forfeitBtn.Text = "Sure?"
		} else {
			s.ws.Send(`{"type":"forfeit"}`)
			s.forfeitBtn.SetHidden(true)
			s.forfeitBtn.SetDisabled(true)
		}
	})
	s.forfeitBtn.BgColor = deepWalnut
	s.forfeitBtn.HoverColor = powderBlush
	s.forfeitBtn.TextColor = white

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

		forfeitClicked := false
		if !s.forfeitBtn.Disabled {
			if s.forfeitBtn.Update() {
				forfeitClicked = true
			}
		}

		if s.myTurn && s.hoverCol >= 0 && !forfeitClicked {
			if inpututil.IsMouseButtonJustReleased(ebiten.MouseButtonLeft) ||
				inpututil.IsKeyJustPressed(ebiten.KeyEnter) ||
				inpututil.IsKeyJustPressed(ebiten.KeyDown) ||
				inpututil.IsKeyJustPressed(ebiten.KeyS) {
				s.placeToken(s.hoverCol)
			}
		}

	case phaseResult:
		s.ticks++
		s.backBtn.Update()
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

func findWinCells(board [6][7]int, player int) [][2]int {
	if player == 0 {
		return nil
	}
	for r := 0; r < 6; r++ {
		for c := 0; c <= 3; c++ {
			if board[r][c] == player && board[r][c+1] == player &&
				board[r][c+2] == player && board[r][c+3] == player {
				return [][2]int{{r, c}, {r, c + 1}, {r, c + 2}, {r, c + 3}}
			}
		}
	}
	for r := 0; r <= 2; r++ {
		for c := 0; c < 7; c++ {
			if board[r][c] == player && board[r+1][c] == player &&
				board[r+2][c] == player && board[r+3][c] == player {
				return [][2]int{{r, c}, {r + 1, c}, {r + 2, c}, {r + 3, c}}
			}
		}
	}
	for r := 0; r <= 2; r++ {
		for c := 0; c <= 3; c++ {
			if board[r][c] == player && board[r+1][c+1] == player &&
				board[r+2][c+2] == player && board[r+3][c+3] == player {
				return [][2]int{{r, c}, {r + 1, c + 1}, {r + 2, c + 2}, {r + 3, c + 3}}
			}
		}
	}
	for r := 3; r < 6; r++ {
		for c := 0; c <= 3; c++ {
			if board[r][c] == player && board[r-1][c+1] == player &&
				board[r-2][c+2] == player && board[r-3][c+3] == player {
				return [][2]int{{r, c}, {r - 1, c + 1}, {r - 2, c + 2}, {r - 3, c + 3}}
			}
		}
	}
	return nil
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
		s.result = session.GameResult{
			Outcome:   p.Outcome,
			WinColor:  p.WinColor,
			ELOBefore: p.ELOBefore,
			ELOAfter:  p.ELOAfter,
			ELODelta:  p.ELODelta,
		}
		var winner int
		if p.WinColor == "red" {
			winner = 1
		} else if p.WinColor == "yellow" {
			winner = 2
		}
		s.winCells = findWinCells(s.cells, winner)
		s.phase = phaseResult
		s.ticks = 0
		s.backBtn.SetHidden(false)
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

	if s.phase == phaseResult {
		for _, cell := range s.winCells {
			r, c := cell[0], cell[1]
			img := s.red
			if s.cells[r][c] == 2 {
				img = s.yel
			}
			imageRow := 5 - r
			tx := boardScreenX + (gridLeft+float64(c)*colStep)*scale
			ty := boardScreenY + (gridTop+float64(imageRow)*rowStep)*scale
			opts := &ebiten.DrawImageOptions{}
			opts.GeoM.Scale(scale, scale)
			opts.GeoM.Translate(tx, ty)
			shine := float32(1.3 + 0.3*math.Sin(float64(s.ticks)*0.1))
			opts.ColorScale.Scale(shine, shine, shine, 1)
			screen.DrawImage(img, opts)
		}

		screen.DrawImage(s.overlay, nil)

		var outcomeText string
		var outcomeColor color.Color
		switch s.result.Outcome {
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

		if s.result.Outcome != "draw" {
			colorText := s.result.WinColor + " wins"
			cb := text.BoundString(basicfont.Face7x13, colorText)
			ctx := (1024 - cb.Dx()) / 2
			text.Draw(screen, colorText, basicfont.Face7x13, ctx, 280, frostedMint)
		}

		delta := s.result.ELOAfter - s.result.ELOBefore
		var eloText string
		var eloColor color.Color
		if delta >= 0 {
			eloText = fmt.Sprintf("%d → %d (+%d)", s.result.ELOBefore, s.result.ELOAfter, delta)
			eloColor = color.RGBA{0x40, 0xff, 0x40, 0xff}
		} else {
			eloText = fmt.Sprintf("%d → %d (%d)", s.result.ELOBefore, s.result.ELOAfter, delta)
			eloColor = color.RGBA{0xff, 0x60, 0x60, 0xff}
		}
		eb := text.BoundString(basicfont.Face7x13, eloText)
		etx := (1024 - eb.Dx()) / 2
		text.Draw(screen, eloText, basicfont.Face7x13, etx, 350, eloColor)

		s.backBtn.Draw(screen)
	}

	if s.phase == phasePlaying && !s.forfeitBtn.Disabled {
		s.forfeitBtn.Draw(screen)
	}
}
