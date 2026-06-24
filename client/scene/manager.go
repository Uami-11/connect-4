//go:build js && wasm

package scene

import (
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text"
	"golang.org/x/image/font/basicfont"

	"connect4/client/net"
	"connect4/client/session"
	"connect4/client/ui"
)

// Scene is implemented by every screen in the game.
type Scene interface {
	Update() error
	Draw(screen *ebiten.Image)
}

// ID identifies each scene, used for the navigation history stack.
type ID int

const (
	IDLogin ID = iota
	IDMenu
	IDMatchmaking
	IDGame
	IDHowToPlay
	IDProfile
	IDProfileOther
	IDLeaderboard
)

// Manager drives the active scene and maintains a navigation history
// stack so the back button always returns to the correct previous screen.
type Manager struct {
	scenes  map[ID]func() Scene // factories, so scenes are fresh each visit
	stack   []ID
	current Scene
	next    *ID // set by a scene to request a transition

	// Challenge overlay state
	challengePollTick int
	challengePollReq  *net.PendingRequest

	challengeActionReq  *net.PendingRequest
	challengeActionType string      // "accept" or "reject"
	challengeFromUser   string

	challengeAcceptBtn *ui.Button
	challengeRejectBtn *ui.Button
}

var (
	challengeBarBg  = color.RGBA{0x0a, 0x0a, 0x1a, 0xdd}
	challengeTextCl = color.RGBA{0xd5, 0xf9, 0xde, 0xff}
	challengeGreen  = color.RGBA{0x44, 0xcc, 0x44, 0xff}
	challengeRed    = color.RGBA{0xff, 0x44, 0x44, 0xff}
)

// NewManager creates a Manager with the given scene factories.
// Start() must be called once before any Update/Draw calls.
func NewManager(factories map[ID]func() Scene) *Manager {
	m := &Manager{
		scenes: factories,
		stack:  []ID{IDLogin},
	}

	m.challengeAcceptBtn = ui.NewButton(630, 58, 100, 30, "Accept", nil)
	m.challengeAcceptBtn.BgColor = challengeGreen
	m.challengeAcceptBtn.TextColor = color.RGBA{0xff, 0xff, 0xff, 0xff}
	m.challengeAcceptBtn.HoverColor = color.RGBA{0x33, 0xaa, 0x33, 0xff}

	m.challengeRejectBtn = ui.NewButton(740, 58, 100, 30, "Reject", nil)
	m.challengeRejectBtn.BgColor = challengeRed
	m.challengeRejectBtn.TextColor = color.RGBA{0xff, 0xff, 0xff, 0xff}
	m.challengeRejectBtn.HoverColor = color.RGBA{0xaa, 0x33, 0x33, 0xff}

	return m
}

// Start creates the initial Login scene. Call once after assigning the
// Manager to its variable so closures in scene constructors see a non-nil
// Manager pointer.
func (m *Manager) Start() {
	m.current = m.scenes[IDLogin]()
}

// Navigate pushes a new scene onto the stack.
func (m *Manager) Navigate(id ID) {
	m.stack = append(m.stack, id)
	m.current = m.scenes[id]()
}

// Back pops the current scene and returns to the previous one.
// Does nothing if already at the root (Login).
func (m *Manager) Back() {
	if len(m.stack) <= 1 {
		return
	}
	m.stack = m.stack[:len(m.stack)-1]
	prev := m.stack[len(m.stack)-1]
	m.current = m.scenes[prev]()
}

// Reset clears the stack and goes to login (used on sign-out).
func (m *Manager) Reset() {
	session.ClearChallenge()
	m.stack = []ID{IDLogin}
	m.current = m.scenes[IDLogin]()
}

// currentID returns the ID of the current (top-of-stack) scene.
func (m *Manager) currentID() ID {
	return m.stack[len(m.stack)-1]
}

// Update advances the current scene and handles challenge overlay input.
func (m *Manager) Update() error {
	// Poll challenge actions (accept/reject response).
	m.pollChallengeAction()

	if err := m.current.Update(); err != nil {
		return err
	}

	// Poll for incoming challenges (only when logged in and not in login).
	if session.Current.LoggedIn && m.currentID() != IDLogin {
		m.pollChallenges()
	}
	return nil
}

// Draw renders the current scene and the challenge overlay (when applicable).
func (m *Manager) Draw(screen *ebiten.Image) {
	m.current.Draw(screen)
	m.drawChallengeOverlay(screen)
}

// --- Challenge overlay ---

func (m *Manager) showChallengeOverlay() bool {
	id := m.currentID()
	return id != IDMatchmaking && id != IDGame && session.CurrentChallenge != nil && session.CurrentChallenge.Status == "pending"
}

func (m *Manager) pollChallengeAction() {
	if m.challengeActionReq == nil {
		return
	}
	net.Poll(m.challengeActionReq)
	if !m.challengeActionReq.Done {
		return
	}
	req := m.challengeActionReq
	m.challengeActionReq = nil

	if req.Err() != nil {
		session.ClearChallenge()
		return
	}

	if m.challengeActionType == "accept" {
		var resp struct {
			Status       string `json:"status,omitempty"`
			OpponentName string `json:"opponent_name,omitempty"`
			YourColor    string `json:"your_color,omitempty"`
			YourTurn     bool   `json:"your_turn,omitempty"`
		}
		if err := net.DecodeResult(req, &resp); err != nil {
			session.ClearChallenge()
			return
		}
		// Match was created. Navigate to matchmaking to connect.
		session.CurrentMatchOpponent = resp.OpponentName
		if resp.YourColor == "yellow" {
			session.CurrentMatchColor = 2
		} else {
			session.CurrentMatchColor = 1
		}
		session.ClearChallenge()
		m.Navigate(IDMatchmaking)
		return
	}

	// reject
	session.ClearChallenge()
}

func (m *Manager) doChallengeAccept() {
	if m.challengeActionReq != nil || session.CurrentChallenge == nil {
		return
	}
	m.challengeActionType = "accept"
	m.challengeFromUser = session.CurrentChallenge.FromUsername
	body := map[string]string{"from_username": m.challengeFromUser}
	m.challengeActionReq = net.StartPostAuth("/challenge/accept", body, session.Current.Token)
}

func (m *Manager) doChallengeReject() {
	if m.challengeActionReq != nil || session.CurrentChallenge == nil {
		return
	}
	m.challengeActionType = "reject"
	m.challengeFromUser = session.CurrentChallenge.FromUsername
	body := map[string]string{"from_username": m.challengeFromUser}
	m.challengeActionReq = net.StartPostAuth("/challenge/reject", body, session.Current.Token)
}

func (m *Manager) pollChallenges() {
	// Every frame: check if an in-flight poll request completed.
	if m.challengePollReq != nil {
		net.Poll(m.challengePollReq)
		if !m.challengePollReq.Done {
			return
		}
		m.handleChallengeResponse()
		return
	}

	// Throttle starting new polls to every 30 frames.
	m.challengePollTick++
	if m.challengePollTick < 30 {
		return
	}
	m.challengePollTick = 0

	// Skip if a challenge or action is already pending.
	if session.CurrentChallenge != nil && session.CurrentChallenge.Status == "pending" {
		return
	}
	if m.challengeActionReq != nil {
		return
	}

	m.challengePollReq = net.StartGet("/challenge/pending", session.Current.Token)
}

func (m *Manager) handleChallengeResponse() {
	req := m.challengePollReq
	m.challengePollReq = nil

	if req.Err() != nil {
		return
	}

	var challenges []struct {
		FromUsername  string `json:"from_username,omitempty"`
		ToUsername    string `json:"to_username,omitempty"`
		Status        string `json:"status"`
		OpponentName  string `json:"opponent_name,omitempty"`
		YourColor     string `json:"your_color,omitempty"`
		YourTurn      bool   `json:"your_turn,omitempty"`
	}
	if err := net.DecodeResult(req, &challenges); err != nil {
		return
	}

	// Check for accepted outgoing challenge → navigate to matchmaking.
	for _, c := range challenges {
		if c.Status == "accepted" && c.OpponentName != "" {
			session.CurrentMatchOpponent = c.OpponentName
			if c.YourColor == "yellow" {
				session.CurrentMatchColor = 2
			} else {
				session.CurrentMatchColor = 1
			}
			session.ClearChallenge()
			m.Navigate(IDMatchmaking)
			return
		}
	}

	// Check for incoming pending challenge.
	for _, c := range challenges {
		if c.Status == "pending" && c.FromUsername != "" {
			session.CurrentChallenge = &session.PendingChallenge{
				FromUsername: c.FromUsername,
				Status:       c.Status,
			}
			return
		}
	}

	// No pending challenges.
	if session.CurrentChallenge != nil && session.CurrentChallenge.Status == "pending" {
		session.ClearChallenge()
	}
}

func (m *Manager) drawChallengeOverlay(screen *ebiten.Image) {
	if !m.showChallengeOverlay() {
		return
	}

	// Black bar background (top-right)
	barX, barY := 620, 10
	barW, barH := 390, 94
	barImg := ebiten.NewImage(barW, barH)
	barImg.Fill(challengeBarBg)
	opts := &ebiten.DrawImageOptions{}
	opts.GeoM.Translate(float64(barX), float64(barY))
	screen.DrawImage(barImg, opts)

	// Text: "{user} requested a battle"
	ch := session.CurrentChallenge
	txt := ch.FromUsername + " requested a battle"
	text.Draw(screen, txt, basicfont.Face7x13, barX+10, barY+28, challengeTextCl)

	// Accept / Reject buttons (only while no action is pending)
	if m.challengeActionReq == nil {
		// Accept
		m.challengeAcceptBtn.X = barX + 10
		m.challengeAcceptBtn.Y = barY + 44
		m.challengeAcceptBtn.SetHidden(false)
		btnClicked := m.challengeAcceptBtn.Update()
		if btnClicked {
			m.doChallengeAccept()
		}
		m.challengeAcceptBtn.Draw(screen)

		// Reject
		m.challengeRejectBtn.X = barX + 120
		m.challengeRejectBtn.Y = barY + 44
		m.challengeRejectBtn.SetHidden(false)
		btnClicked = m.challengeRejectBtn.Update()
		if btnClicked {
			m.doChallengeReject()
		}
		m.challengeRejectBtn.Draw(screen)
	} else {
		lt := "loading..."
		text.Draw(screen, lt, basicfont.Face7x13, barX+10, barY+64, challengeTextCl)
	}
}
