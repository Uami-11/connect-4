//go:build js && wasm

package scene

import (
	"image/color"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hajimehoshi/ebiten/v2/text"
	"golang.org/x/image/font/basicfont"

	"connect4/client/assets"
	"connect4/client/net"
	"connect4/client/session"
	"connect4/client/ui"
)

var (
	airForceBlue = color.RGBA{0x66, 0x89, 0xa1, 0xff}
	frostedMint  = color.RGBA{0xd5, 0xf9, 0xde, 0xff}
	deepWalnut   = color.RGBA{0x53, 0x3e, 0x2d, 0xff}
	powderBlush  = color.RGBA{0xed, 0xb6, 0xa3, 0xff}
	darkCyan     = color.RGBA{0x11, 0x9d, 0xa4, 0xff}
	white        = color.RGBA{0xff, 0xff, 0xff, 0xff}
	lightGray    = color.RGBA{0xcc, 0xcc, 0xcc, 0xff}
)

type Login struct {
	mgr *Manager
	bg  *ebiten.Image

	mode           loginMode
	usernameInput  *ui.Input
	passwordInput  *ui.Input
	confirmInput   *ui.Input
	loginBtn       *ui.Button
	registerBtn    *ui.Button
	errMsg         string
	loading        bool

	btnW, btnH int
}

type loginMode int

const (
	modeLogin loginMode = iota
	modeRegister
)

func NewLogin(mgr *Manager) *Login {
	boxW, boxH := 400, 380
	boxX := (1024 - boxW) / 2
	boxY := (768 - boxH) / 2
	btnW, btnH := 200, 44

	inW := 320
	inH := 32
	inX := boxX + (boxW-inW)/2
	rowGap := 48
	startY := boxY + 60

	s := &Login{
		mgr: mgr,
		bg:  assets.MustLoadImage("images/backgrounds/login.png"),
		mode: modeLogin,
		usernameInput: ui.NewInput(inX, startY, inW, inH, "username"),
		passwordInput: ui.NewInput(inX, startY+rowGap, inW, inH, "password"),
		confirmInput:  ui.NewInput(inX, startY+rowGap*2, inW, inH, "confirm password"),
		btnW:          btnW,
		btnH:          btnH,
	}
	s.usernameInput.TextColor = deepWalnut
	s.passwordInput.TextColor = deepWalnut
	s.passwordInput.Secret = true
	s.confirmInput.TextColor = deepWalnut
	s.confirmInput.Secret = true

	btnX := boxX + (boxW-btnW)/2
	btnY := startY + rowGap*2 + 30

	s.loginBtn = ui.NewButton(btnX, btnY, btnW, btnH, "Log In", func() {
		s.submit()
	})
	s.loginBtn.BgColor = deepWalnut
	s.loginBtn.HoverColor = darkCyan
	s.loginBtn.TextColor = white

	s.registerBtn = ui.NewButton(btnX, btnY, btnW, btnH, "Register", func() {
		s.submit()
	})
	s.registerBtn.BgColor = deepWalnut
	s.registerBtn.HoverColor = darkCyan
	s.registerBtn.TextColor = white
	s.registerBtn.SetHidden(true)

	return s
}

func (s *Login) Update() error {
	s.usernameInput.Update()
	s.passwordInput.Update()
	s.confirmInput.Update()
	s.loginBtn.Update()
	s.registerBtn.Update()

	// Toggle mode on link click
	if inpututil.IsMouseButtonJustReleased(ebiten.MouseButtonLeft) {
		mx, my := ebiten.CursorPosition()
		toggleBounds := s.toggleBounds()
		if mx >= toggleBounds[0] && mx <= toggleBounds[2] &&
			my >= toggleBounds[1] && my <= toggleBounds[3] {
			s.toggleMode()
		}
	}

	// Enter submits
	if inpututil.IsKeyJustPressed(ebiten.KeyEnter) {
		if s.mode == modeRegister {
			s.confirmInput.Focus()
		}
		s.submit()
	}

	// Tab / DownArrow cycles focus
	if inpututil.IsKeyJustPressed(ebiten.KeyTab) || inpututil.IsKeyJustPressed(ebiten.KeyDown) {
		s.cycleFocus()
	}

	return nil
}

func (s *Login) cycleFocus() {
	if s.usernameInput.IsFocused() {
		s.usernameInput.Blur()
		s.passwordInput.Focus()
	} else if s.passwordInput.IsFocused() {
		s.passwordInput.Blur()
		if s.mode == modeRegister {
			s.confirmInput.Focus()
		} else {
			s.usernameInput.Focus()
		}
	} else if s.confirmInput.IsFocused() {
		s.confirmInput.Blur()
		s.usernameInput.Focus()
	} else {
		s.usernameInput.Focus()
	}
}

func (s *Login) toggleMode() {
	s.errMsg = ""
	s.usernameInput.Clear()
	s.passwordInput.Clear()
	s.confirmInput.Clear()
	if s.mode == modeLogin {
		s.mode = modeRegister
		s.loginBtn.SetHidden(true)
		s.registerBtn.SetHidden(false)
	} else {
		s.mode = modeLogin
		s.loginBtn.SetHidden(false)
		s.registerBtn.SetHidden(true)
	}
}

func (s *Login) submit() {
	if s.loading {
		return
	}
	s.errMsg = ""

	username := strings.TrimSpace(s.usernameInput.Text)
	password := s.passwordInput.Text

	if username == "" || password == "" {
		s.errMsg = "fill in all fields"
		return
	}

	if s.mode == modeRegister {
		confirm := s.confirmInput.Text
		if password != confirm {
			s.errMsg = "passwords do not match"
			return
		}
	}

	s.loading = true
	go func() {
		var resp struct {
			Token    string `json:"token"`
			Username string `json:"username"`
			ELO      int    `json:"elo"`
		}
		var err error
		if s.mode == modeLogin {
			err = net.Post("/login", map[string]string{
				"username": username,
				"password": password,
			}, &resp)
		} else {
			err = net.Post("/register", map[string]string{
				"username": username,
				"password": password,
			}, &resp)
		}

		if err != nil {
			s.errMsg = err.Error()
			s.loading = false
			return
		}

		session.Current.Token = resp.Token
		session.Current.Username = resp.Username
		session.Current.ELO = resp.ELO
		session.Current.LoggedIn = true
		s.loading = false
		s.mgr.Navigate(IDMenu)
	}()
}

func (s *Login) toggleBounds() [4]int {
	boxW := 400
	boxX := (1024 - boxW) / 2
	boxY := (768 - 380) / 2
	btnY := boxY + 60 + 48*2 + 30 + 44 + 8
	txt := "register instead?"
	if s.mode == modeRegister {
		txt = "login instead?"
	}
	b := text.BoundString(basicfont.Face7x13, txt)
	tx := boxX + (boxW-b.Dx())/2
	return [4]int{tx, btnY, tx + b.Dx(), btnY + b.Dy()}
}

func (s *Login) Draw(screen *ebiten.Image) {
	screen.DrawImage(s.bg, nil)

	boxW, boxH := 400, 380
	boxX := (1024 - boxW) / 2
	boxY := (768 - boxH) / 2

	// Semi-transparent overlay box
	boxImg := ebiten.NewImage(boxW, boxH)
	boxImg.Fill(color.RGBA{0x0a, 0x0a, 0x1a, 0xaa})
	opts := &ebiten.DrawImageOptions{}
	opts.GeoM.Translate(float64(boxX), float64(boxY))
	screen.DrawImage(boxImg, opts)

	// Title
	title := "Log In"
	if s.mode == modeRegister {
		title = "Register"
	}
	tx := boxX + (boxW-text.BoundString(basicfont.Face7x13, title).Dx())/2
	text.Draw(screen, title, basicfont.Face7x13, tx, boxY+30, frostedMint)

	// Inputs
	s.usernameInput.Draw(screen)
	s.passwordInput.Draw(screen)
	if s.mode == modeRegister {
		s.confirmInput.Draw(screen)
	}

	// Buttons
	s.loginBtn.Draw(screen)
	s.registerBtn.Draw(screen)

	// Toggle link
	toggleTxt := "register instead?"
	if s.mode == modeRegister {
		toggleTxt = "login instead?"
	}
	b := text.BoundString(basicfont.Face7x13, toggleTxt)
	btnY := boxY + 60 + 48*2 + 30 + 44 + 8
	tx = boxX + (boxW-b.Dx())/2
	text.Draw(screen, toggleTxt, basicfont.Face7x13, tx, btnY, powderBlush)

	// Error message
	if s.errMsg != "" {
		eb := text.BoundString(basicfont.Face7x13, s.errMsg)
		ex := boxX + (boxW-eb.Dx())/2
		text.Draw(screen, s.errMsg, basicfont.Face7x13, ex, boxY+boxH-20, color.RGBA{0xff, 0x60, 0x60, 0xff})
	}

	// Loading
	if s.loading {
		lt := "loading..."
		lb := text.BoundString(basicfont.Face7x13, lt)
		lx := boxX + (boxW-lb.Dx())/2
		text.Draw(screen, lt, basicfont.Face7x13, lx, boxY+boxH-40, lightGray)
	}
}
