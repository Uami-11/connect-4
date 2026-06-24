//go:build js && wasm

package scene

import (
	"image/color"
	"math"
	"math/rand"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hajimehoshi/ebiten/v2/text"
	"golang.org/x/image/font/basicfont"

	"connect4/client/assets"
	"connect4/client/ui"
)

const (
	tutorialScale        = 0.5
	tutorialBoardX       = 30
	tutorialBoardY       = 135
	tutorialColStep      = 153
	tutorialRowStep      = 151
	tutorialGridLeft     = 24
	tutorialGridTop      = 67
	tutorialHoverTokenY  = 80
	tutorialAIDelay      = 30
)

type tutorialPhase int

const (
	tutPhasePlaying tutorialPhase = iota
	tutPhasePause
)

type HowToPlay struct {
	mgr *Manager
	bg  *ebiten.Image
	brd *ebiten.Image
	red *ebiten.Image
	yel *ebiten.Image

	phase tutorialPhase

	board  [6][7]int
	myTurn bool
	aiTimer int

	hoverCol       int
	keyboardActive bool
	lastMouseCol   int

	winCells [][2]int
	ticks    int

	backBtn *ui.Button
}

func NewHowToPlay(mgr *Manager) *HowToPlay {
	s := &HowToPlay{
		mgr:          mgr,
		bg:           assets.MustLoadImage("images/backgrounds/game.png"),
		brd:          assets.MustLoadImage("images/main/board.png"),
		red:          assets.MustLoadImage("images/main/red_token.png"),
		yel:          assets.MustLoadImage("images/main/yellow_token.png"),
		phase:        tutPhasePlaying,
		myTurn:       true,
		hoverCol:     -1,
		lastMouseCol: -1,
	}

	s.backBtn = ui.NewButton(20, 704, 160, 44, "Back to Menu", func() {
		mgr.Navigate(IDMenu)
	})
	s.backBtn.BgColor = deepWalnut
	s.backBtn.HoverColor = darkCyan
	s.backBtn.TextColor = white

	return s
}

func (s *HowToPlay) Update() error {
	s.backBtn.Update()

	switch s.phase {
	case tutPhasePlaying:
		if s.aiTimer > 0 {
			s.aiTimer--
			break
		}

		if !s.myTurn {
			s.aiMove()
			break
		}

		mx, _ := ebiten.CursorPosition()
		scaledLeft := float64(tutorialBoardX) + float64(tutorialGridLeft)*tutorialScale
		scaledColW := float64(tutorialColStep) * tutorialScale
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
			if (inpututil.IsMouseButtonJustReleased(ebiten.MouseButtonLeft) && !s.backBtn.IsHovered()) ||
				inpututil.IsKeyJustPressed(ebiten.KeyEnter) ||
				inpututil.IsKeyJustPressed(ebiten.KeyDown) ||
				inpututil.IsKeyJustPressed(ebiten.KeyS) {
				s.playerDrop(s.hoverCol)
			}
		}

	case tutPhasePause:
		s.ticks++
	}

	return nil
}

func (s *HowToPlay) playerDrop(col int) {
	row := tutorialDrop(&s.board, col, 1)
	if row < 0 {
		return
	}

	if tutorialCheckWin(s.board, 1) {
		s.winCells = findWinCells(s.board, 1)
		s.phase = tutPhasePause
		s.ticks = 0
		return
	}
	if tutorialIsFull(s.board) {
		s.phase = tutPhasePause
		s.ticks = 0
		return
	}

	s.myTurn = false
	s.aiTimer = tutorialAIDelay
}

func (s *HowToPlay) aiMove() {
	cols := make([]int, 0, 7)
	for c := 0; c < 7; c++ {
		if s.board[5][c] == 0 {
			cols = append(cols, c)
		}
	}
	if len(cols) == 0 {
		return
	}
	col := cols[rand.Intn(len(cols))]
	tutorialDrop(&s.board, col, 2)

	if tutorialCheckWin(s.board, 2) {
		s.winCells = findWinCells(s.board, 2)
		s.phase = tutPhasePause
		s.ticks = 0
		return
	}
	if tutorialIsFull(s.board) {
		s.phase = tutPhasePause
		s.ticks = 0
		return
	}

	s.myTurn = true
}

func (s *HowToPlay) Draw(screen *ebiten.Image) {
	screen.DrawImage(s.bg, nil)

	opts := &ebiten.DrawImageOptions{}
	opts.GeoM.Scale(tutorialScale, tutorialScale)
	opts.GeoM.Translate(tutorialBoardX, tutorialBoardY)
	screen.DrawImage(s.brd, opts)

	for r := 0; r < 6; r++ {
		for c := 0; c < 7; c++ {
			player := s.board[r][c]
			if player == 0 {
				continue
			}
			img := s.red
			if player == 2 {
				img = s.yel
			}
			imageRow := 5 - r
			tx := float64(tutorialBoardX) + (float64(tutorialGridLeft)+float64(c)*tutorialColStep)*tutorialScale
			ty := float64(tutorialBoardY) + (float64(tutorialGridTop)+float64(imageRow)*tutorialRowStep)*tutorialScale
			opts := &ebiten.DrawImageOptions{}
			opts.GeoM.Scale(tutorialScale, tutorialScale)
			opts.GeoM.Translate(tx, ty)
			screen.DrawImage(img, opts)
		}
	}

	// Hover token
	if s.phase == tutPhasePlaying && s.hoverCol >= 0 && s.myTurn {
		hx := float64(tutorialBoardX) + (float64(tutorialGridLeft)+float64(s.hoverCol)*tutorialColStep)*tutorialScale
		opts := &ebiten.DrawImageOptions{}
		opts.GeoM.Scale(tutorialScale, tutorialScale)
		opts.GeoM.Translate(hx, tutorialHoverTokenY)
		screen.DrawImage(s.red, opts)
	}

	// Pulsing win tokens
	if s.phase == tutPhasePause {
		for _, cell := range s.winCells {
			r, c := cell[0], cell[1]
			img := s.red
			if s.board[r][c] == 2 {
				img = s.yel
			}
			imageRow := 5 - r
			tx := float64(tutorialBoardX) + (float64(tutorialGridLeft)+float64(c)*tutorialColStep)*tutorialScale
			ty := float64(tutorialBoardY) + (float64(tutorialGridTop)+float64(imageRow)*tutorialRowStep)*tutorialScale
			opts := &ebiten.DrawImageOptions{}
			opts.GeoM.Scale(tutorialScale, tutorialScale)
			opts.GeoM.Translate(tx, ty)
			shine := float32(1.3 + 0.3*math.Sin(float64(s.ticks)*0.1))
			opts.ColorScale.Scale(shine, shine, shine, 1)
			screen.DrawImage(img, opts)
		}
	}

	// Instructions
	if s.phase == tutPhasePlaying || s.phase == tutPhasePause {
		ix := 590
		s.drawScaledText(screen, "How to Play", float64(ix), 150, 3, frostedMint)

		lines := []string{
			"\u2022 Click a column",
			"  to drop your token",
			"",
			"\u2022 Arrow keys or",
			"  W A S D to move",
			"",
			"\u2022 Enter, Down,",
			"  or S to drop",
			"",
			"\u2022 Get 4 in a row",
			"  to win!",
		}
		ly := 240.0
		for _, line := range lines {
			s.drawScaledText(screen, line, float64(ix), ly, 1.7, frostedMint)
			ly += 30
		}
	}

	s.backBtn.Draw(screen)
}

func (s *HowToPlay) drawScaledText(screen *ebiten.Image, str string, x, y, scale float64, clr color.Color) {
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

// tutorial helpers

func tutorialDrop(board *[6][7]int, col, player int) int {
	if col < 0 || col >= 7 {
		return -1
	}
	for r := 0; r < 6; r++ {
		if board[r][col] == 0 {
			board[r][col] = player
			return r
		}
	}
	return -1
}

func tutorialCheckWin(board [6][7]int, player int) bool {
	for r := 0; r < 6; r++ {
		for c := 0; c <= 3; c++ {
			if board[r][c] == player && board[r][c+1] == player &&
				board[r][c+2] == player && board[r][c+3] == player {
				return true
			}
		}
	}
	for r := 0; r <= 2; r++ {
		for c := 0; c < 7; c++ {
			if board[r][c] == player && board[r+1][c] == player &&
				board[r+2][c] == player && board[r+3][c] == player {
				return true
			}
		}
	}
	for r := 0; r <= 2; r++ {
		for c := 0; c <= 3; c++ {
			if board[r][c] == player && board[r+1][c+1] == player &&
				board[r+2][c+2] == player && board[r+3][c+3] == player {
				return true
			}
		}
	}
	for r := 3; r < 6; r++ {
		for c := 0; c <= 3; c++ {
			if board[r][c] == player && board[r-1][c+1] == player &&
				board[r-2][c+2] == player && board[r-3][c+3] == player {
				return true
			}
		}
	}
	return false
}

func tutorialIsFull(board [6][7]int) bool {
	for c := 0; c < 7; c++ {
		if board[5][c] == 0 {
			return false
		}
	}
	return true
}
