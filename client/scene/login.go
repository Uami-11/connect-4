//go:build js && wasm

package scene

import (
	"github.com/hajimehoshi/ebiten/v2"

	"connect4/client/assets"
)

// Login is the login and registration screen.
// It toggles between a login form and a register form.
type Login struct {
	mgr        *Manager
	bg         *ebiten.Image
	mode       loginMode // modeLogin or modeRegister
	username   string
	password   string
	confirm    string    // only used in register mode
	errMsg     string
	loading    bool
}

type loginMode int

const (
	modeLogin    loginMode = iota
	modeRegister
)

// NewLogin creates the Login scene.
func NewLogin(mgr *Manager) *Login {
	return &Login{
		mgr:  mgr,
		bg:   assets.MustLoadImage("images/backgrounds/login.png"),
		mode: modeLogin,
	}
}

func (s *Login) Update() error {
	// TODO: handle keyboard input for username/password fields,
	// login/register button clicks, toggle between modes.
	// On success: populate session.Current and mgr.Navigate(IDMenu).
	return nil
}

func (s *Login) Draw(screen *ebiten.Image) {
	// TODO: draw bg, input boxes, buttons, error message.
	opts := &ebiten.DrawImageOptions{}
	screen.DrawImage(s.bg, opts)
}
