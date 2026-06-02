//go:build js && wasm

package assets

import (
	"bytes"
	_ "embed"
	"fmt"
	"image"
	_ "image/png"

	"github.com/hajimehoshi/ebiten/v2"
)

//go:embed images/backgrounds/login.png
var bgLogin []byte

//go:embed images/backgrounds/main_menu.png
var bgMainMenu []byte

//go:embed images/backgrounds/matchmaking.png
var bgMatchmaking []byte

//go:embed images/backgrounds/game.png
var bgGame []byte

//go:embed images/backgrounds/leaderboard.png
var bgLeaderboard []byte

//go:embed images/backgrounds/user_info.png
var bgUserInfo []byte

//go:embed images/main/board.png
var imgBoard []byte

//go:embed images/main/red_token.png
var imgRedToken []byte

//go:embed images/main/yellow_token.png
var imgYellowToken []byte

//go:embed images/main/socket.png
var imgSocket []byte

//go:embed images/main/plug.png
var imgPlug []byte

//go:embed "images/main/sockets disconnection .png"
var imgSocketsDisconnection []byte

var cache = make(map[string]*ebiten.Image)

// MustLoadImage loads a previously embedded image by its relative path
// under assets/images/. Panics if the path is unknown or the PNG is invalid.
func MustLoadImage(path string) *ebiten.Image {
	if img, ok := cache[path]; ok {
		return img
	}

	data, ok := lookup(path)
	if !ok {
		panic(fmt.Sprintf("assets: unknown image path %q", path))
	}

	raw, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		panic(fmt.Sprintf("assets: decoding %q: %v", path, err))
	}

	img := ebiten.NewImageFromImage(raw)
	cache[path] = img
	return img
}

// lookup returns the embedded bytes for a known image path.
func lookup(path string) ([]byte, bool) {
	m := map[string][]byte{
		"images/backgrounds/login.png":         bgLogin,
		"images/backgrounds/main_menu.png":     bgMainMenu,
		"images/backgrounds/matchmaking.png":   bgMatchmaking,
		"images/backgrounds/game.png":          bgGame,
		"images/backgrounds/leaderboard.png":   bgLeaderboard,
		"images/backgrounds/user_info.png":     bgUserInfo,
		"images/main/board.png":                imgBoard,
		"images/main/red_token.png":            imgRedToken,
		"images/main/yellow_token.png":         imgYellowToken,
		"images/main/socket.png":               imgSocket,
		"images/main/plug.png":                 imgPlug,
		"images/main/sockets disconnection .png": imgSocketsDisconnection,
	}
	b, ok := m[path]
	return b, ok
}
