//go:build js && wasm

package scene

import (
	"encoding/json"
	"image/color"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text"
	"golang.org/x/image/font/basicfont"

	"connect4/client/assets"
	"connect4/client/net"
	"connect4/client/session"
	"connect4/client/ui"
)

var (
	matchTextColor  = color.RGBA{0xd5, 0xf9, 0xde, 0xff}
	matchDimColor   = color.RGBA{0xcc, 0xcc, 0xcc, 0xff}
	matchErrColor   = color.RGBA{0xff, 0x60, 0x60, 0xff}
)

type matchState int

const (
	matchConnecting matchState = iota
	matchAuthing
	matchQueued
	matchCancelling
	matchMatched
	matchError
)

// Matchmaking shows a searching animation and waits for a match.
// The player can cancel at any time before being paired.
type Matchmaking struct {
	mgr    *Manager
	bg     *ebiten.Image
	ws     *net.WSConn
	state  matchState
	errMsg string

	cancelBtn *ui.Button
	backBtn   *ui.Button

	startTime time.Time
}

// NewMatchmaking creates the Matchmaking scene.
func NewMatchmaking(mgr *Manager) *Matchmaking {
	s := &Matchmaking{
		mgr:       mgr,
		bg:        assets.MustLoadImage("images/backgrounds/matchmaking.png"),
		state:     matchConnecting,
		startTime: time.Now(),
	}

	btnW, btnH := 200, 44
	btnX := (1024 - btnW) / 2
	btnY := 600

	s.cancelBtn = ui.NewButton(btnX, btnY, btnW, btnH, "Cancel", func() {
		s.doCancel()
	})
	s.cancelBtn.BgColor = color.RGBA{0x53, 0x3e, 0x2d, 0xff}
	s.cancelBtn.HoverColor = color.RGBA{0x11, 0x9d, 0xa4, 0xff}
	s.cancelBtn.TextColor = color.RGBA{0xff, 0xff, 0xff, 0xff}

	s.backBtn = ui.NewButton(btnX, btnY, btnW, btnH, "Back to Menu", func() {
		s.mgr.Back()
	})
	s.backBtn.BgColor = color.RGBA{0x53, 0x3e, 0x2d, 0xff}
	s.backBtn.HoverColor = color.RGBA{0x11, 0x9d, 0xa4, 0xff}
	s.backBtn.TextColor = color.RGBA{0xff, 0xff, 0xff, 0xff}
	s.backBtn.SetHidden(true)

	ws, err := net.NewWSConn()
	if err != nil {
		s.state = matchError
		s.errMsg = "failed to create connection"
		return s
	}
	s.ws = ws

	return s
}

func (s *Matchmaking) doCancel() {
	switch s.state {
	case matchConnecting, matchAuthing, matchError:
		if s.ws != nil {
			s.ws.Close()
		}
		s.mgr.Back()
	case matchQueued:
		s.ws.Send(`{"type":"cancel"}`)
		s.state = matchCancelling
	default:
		// Already cancelling, matched — ignore.
	}
}

func (s *Matchmaking) Update() error {
	// Detect connection loss while connected.
	if s.ws != nil && s.state != matchConnecting && s.state != matchError {
		select {
		case <-s.ws.Done():
			s.state = matchError
			s.errMsg = "connection lost"
		default:
		}
	}

	// Poll connection state.
	if s.ws != nil && s.state == matchConnecting {
		if s.ws.IsOpen() {
			s.state = matchAuthing
			s.ws.Send(`{"type":"auth","token":"` + session.Current.Token + `"}`)
		} else if time.Since(s.startTime) > 10*time.Second {
			s.state = matchError
			s.errMsg = "connection timeout"
		}
	}

	// Send queue message after auth (will be read in order by server).
	if s.ws != nil && s.state == matchAuthing {
		s.ws.Send(`{"type":"queue"}`)
		s.state = matchQueued
	}

	// Drain the receive channel.
	if s.ws != nil {
		select {
		case msg := <-s.ws.Recv():
			s.handleMessage(msg)
		default:
		}
	}

	// Handle cancel response — go back when server confirms.
	if s.state == matchCancelling {
		select {
		case msg := <-s.ws.Recv():
			s.handleMessage(msg)
			if s.state != matchCancelling {
				return nil
			}
		default:
		}
	}

	// Update buttons.
	if s.state == matchError {
		s.backBtn.Update()
	} else {
		s.cancelBtn.Update()
	}

	return nil
}

func (s *Matchmaking) handleMessage(raw string) {
	var header struct {
		Type    string          `json:"type"`
		Payload json.RawMessage `json:"payload"`
	}
	if err := json.Unmarshal([]byte(raw), &header); err != nil {
		return
	}

	switch header.Type {
	case "waiting":
		// Server acknowledged queue join — we're waiting.
		if s.state == matchQueued {
			// Already in queued state.
		}

	case "cancelled":
		// Server confirmed cancel — go back to menu.
		if s.ws != nil {
			s.ws.Close()
		}
		s.mgr.Back()

	case "matched":
		var payload struct {
			OpponentName string `json:"opponent_name"`
			YourColor    string `json:"your_color"`
			YourTurn     bool   `json:"your_turn"`
		}
		if err := json.Unmarshal(header.Payload, &payload); err != nil {
			s.state = matchError
			s.errMsg = "invalid match data"
			return
		}
		// Pass the WS connection to the game scene.
		s.state = matchMatched
		session.CurrentWS = s.ws
		s.ws = nil // prevent closing it on scene exit
		s.mgr.Navigate(IDGame)

	case "error":
		var payload struct {
			Message string `json:"message"`
		}
		if err := json.Unmarshal(header.Payload, &payload); err != nil || payload.Message == "" {
			s.state = matchError
			s.errMsg = "server error"
		} else {
			s.state = matchError
			s.errMsg = payload.Message
		}
	}
}

func (s *Matchmaking) Draw(screen *ebiten.Image) {
	screen.DrawImage(s.bg, nil)

	var statusText string
	var statusColor color.Color

	switch s.state {
	case matchError:
		statusText = "Error: " + s.errMsg
		statusColor = matchErrColor
	case matchCancelling:
		statusText = "cancelling..."
		statusColor = matchDimColor
	case matchMatched:
		statusText = "matched! starting game..."
		statusColor = matchTextColor
	case matchQueued:
		statusText = "searching for opponent..."
		statusColor = matchTextColor
	case matchAuthing:
		statusText = "joining queue..."
		statusColor = matchTextColor
	default:
		statusText = "connecting..."
		statusColor = matchDimColor
	}

	b := text.BoundString(basicfont.Face7x13, statusText)
	tx := (1024 - b.Dx()) / 2
	ty := 350
	text.Draw(screen, statusText, basicfont.Face7x13, tx, ty, statusColor)

	// Draw buttons.
	if s.state == matchError {
		s.backBtn.Draw(screen)
	} else {
		s.cancelBtn.Draw(screen)
	}
}
