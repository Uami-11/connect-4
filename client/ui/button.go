//go:build js && wasm

package ui

import (
	"image"
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hajimehoshi/ebiten/v2/text"
	"golang.org/x/image/font/basicfont"
)

var (
	btnFace = basicfont.Face7x13
)

type Button struct {
	X, Y, W, H    int
	Text          string
	TextColor     color.Color
	BgColor       color.Color
	HoverColor    color.Color
	DisabledColor color.Color
	Disabled      bool
	hidden        bool
	onClick       func()
}

func NewButton(x, y, w, h int, text string, onClick func()) *Button {
	return &Button{
		X:             x,
		Y:             y,
		W:             w,
		H:             h,
		Text:          text,
		TextColor:     color.RGBA{0xff, 0xff, 0xff, 0xff},
		BgColor:       color.RGBA{0x53, 0x3e, 0x2d, 0xff},
		HoverColor:    color.RGBA{0x66, 0x89, 0xa1, 0xff},
		DisabledColor: color.RGBA{0x88, 0x88, 0x88, 0xff},
		onClick:       onClick,
	}
}

func (b *Button) SetHidden(h bool) {
	b.hidden = h
}

func (b *Button) SetDisabled(d bool) {
	b.Disabled = d
}

func (b *Button) Update() bool {
	if b.hidden || b.Disabled {
		return false
	}
	if inpututil.IsMouseButtonJustReleased(ebiten.MouseButtonLeft) {
		mx, my := ebiten.CursorPosition()
		if mx >= b.X && mx < b.X+b.W && my >= b.Y && my < b.Y+b.H {
			if b.onClick != nil {
				b.onClick()
			}
			return true
		}
	}
	return false
}

func (b *Button) IsHovered() bool {
	if b.hidden {
		return false
	}
	mx, my := ebiten.CursorPosition()
	return image.Rect(b.X, b.Y, b.X+b.W, b.Y+b.H).Overlaps(image.Rect(mx, my, mx+1, my+1))
}

func (b *Button) Draw(screen *ebiten.Image) {
	if b.hidden {
		return
	}
	bg := b.BgColor
	if b.Disabled {
		bg = b.DisabledColor
	} else if b.IsHovered() {
		bg = b.HoverColor
	}

	ebitenutilDrawRect(screen, b.X, b.Y, b.W, b.H, bg)

	bounds := text.BoundString(btnFace, b.Text)
	tx := b.X + (b.W-bounds.Dx())/2
	ty := b.Y + (b.H-bounds.Dy())/2 + bounds.Dy()
	text.Draw(screen, b.Text, btnFace, tx, ty, b.TextColor)
}

func ebitenutilDrawRect(screen *ebiten.Image, x, y, w, h int, clr color.Color) {
	img := ebiten.NewImage(w, h)
	img.Fill(clr)
	opts := &ebiten.DrawImageOptions{}
	opts.GeoM.Translate(float64(x), float64(y))
	screen.DrawImage(img, opts)
}
