//go:build js && wasm

package ui

import (
	"image"
	"image/color"
	"strings"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hajimehoshi/ebiten/v2/text"
	"golang.org/x/image/font/basicfont"
)

var inputFace = basicfont.Face7x13

type Input struct {
	X, Y, W, H  int
	Placeholder string
	Secret      bool // password mode
	Text        string
	MaxLen      int

	BgColor    color.Color
	TextColor  color.Color
	FocusColor color.Color

	focused  bool
	blinkOn  bool
	lastBlink time.Time
}

func NewInput(x, y, w, h int, placeholder string) *Input {
	return &Input{
		X:           x,
		Y:           y,
		W:           w,
		H:           h,
		Placeholder: placeholder,
		MaxLen:      30,
		BgColor:     color.RGBA{0xff, 0xff, 0xff, 0xdd},
		TextColor:   color.RGBA{0x53, 0x3e, 0x2d, 0xff},
		FocusColor:  color.RGBA{0x11, 0x9d, 0xa4, 0xff},
		lastBlink:   time.Now(),
	}
}

func (s *Input) Focus() {
	s.focused = true
}

func (s *Input) Blur() {
	s.focused = false
}

func (s *Input) IsFocused() bool {
	return s.focused
}

func (s *Input) IsHovered() bool {
	mx, my := ebiten.CursorPosition()
	return image.Rect(s.X, s.Y, s.X+s.W, s.Y+s.H).Overlaps(image.Rect(mx, my, mx+1, my+1))
}

func (s *Input) Update() {
	if inpututil.IsMouseButtonJustReleased(ebiten.MouseButtonLeft) {
		s.focused = s.IsHovered()
	}

	if !s.focused {
		return
	}

	// Cursor blink
	if time.Since(s.lastBlink) > 530*time.Millisecond {
		s.blinkOn = !s.blinkOn
		s.lastBlink = time.Now()
	}

	// Typing
	for _, ch := range ebiten.InputChars() {
		if len(s.Text) < s.MaxLen {
			s.Text += string(ch)
		}
	}

	// Backspace
	if inpututil.IsKeyJustPressed(ebiten.KeyBackspace) && len(s.Text) > 0 {
		s.Text = s.Text[:len(s.Text)-1]
	}
}

func (s *Input) Clear() {
	s.Text = ""
}

func (s *Input) Draw(screen *ebiten.Image) {
	bg := s.BgColor
	border := s.TextColor
	if s.focused {
		border = s.FocusColor
	}

	ebitenutilDrawRect(screen, s.X, s.Y, s.W, s.H, bg)

	// Border lines
	borderImg := ebiten.NewImage(1, 1)
	borderImg.Fill(border)
	opts := &ebiten.DrawImageOptions{}
	// top
	opts.GeoM.Translate(float64(s.X), float64(s.Y))
	screen.DrawImage(borderImg, scaleOpts(opts, s.W, 1))
	// bottom
	opts.GeoM.Reset()
	opts.GeoM.Translate(float64(s.X), float64(s.Y+s.H-1))
	screen.DrawImage(borderImg, scaleOpts(opts, s.W, 1))
	// left
	opts.GeoM.Reset()
	opts.GeoM.Translate(float64(s.X), float64(s.Y))
	screen.DrawImage(borderImg, scaleOpts(opts, 1, s.H))
	// right
	opts.GeoM.Reset()
	opts.GeoM.Translate(float64(s.X+s.W-1), float64(s.Y))
	screen.DrawImage(borderImg, scaleOpts(opts, 1, s.H))

	// Text
	displayText := s.Text
	if s.Secret {
		displayText = strings.Repeat("*", len(s.Text))
	}

	padding := 6
	tx := s.X + padding
	ty := s.Y + (s.H-inputFace.Height)/2 + inputFace.Ascent

	if len(displayText) == 0 && !s.focused {
		text.Draw(screen, s.Placeholder, inputFace, tx, ty, color.RGBA{0x88, 0x88, 0x88, 0xff})
	} else {
		text.Draw(screen, displayText, inputFace, tx, ty, s.TextColor)
	}

	// Cursor
	if s.focused && s.blinkOn {
		cursorX := tx + text.BoundString(inputFace, displayText).Dx()
		ebitenutilDrawRect(screen, cursorX, s.Y+4, 2, s.H-8, s.TextColor)
	}
}

func scaleOpts(opts *ebiten.DrawImageOptions, w, h int) *ebiten.DrawImageOptions {
	opts.GeoM.Scale(float64(w), float64(h))
	return opts
}
