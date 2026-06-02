//go:build js && wasm

package main

import (
	"log"

	"github.com/hajimehoshi/ebiten/v2"

	"connect4/client/scene"
)

const (
	screenW = 1024
	screenH = 768
)

// game is the top-level Ebitengine game struct.
type game struct {
	mgr *scene.Manager
}

func (g *game) Update() error {
	return g.mgr.Update()
}

func (g *game) Draw(screen *ebiten.Image) {
	g.mgr.Draw(screen)
}

func (g *game) Layout(_, _ int) (int, int) {
	return screenW, screenH
}

func main() {
	ebiten.SetWindowSize(screenW, screenH)
	ebiten.SetWindowTitle("Connect 4")

	mgr := buildManager()

	if err := ebiten.RunGame(&game{mgr: mgr}); err != nil {
		log.Fatal(err)
	}
}

// buildManager wires all scene constructors into the manager.
// Each factory is called fresh every time that scene is navigated to,
// so scenes always start with clean state.
func buildManager() *scene.Manager {
	var mgr *scene.Manager

	factories := map[scene.ID]func() scene.Scene{
		scene.IDLogin:        func() scene.Scene { return scene.NewLogin(mgr) },
		scene.IDMenu:         func() scene.Scene { return scene.NewMenu(mgr) },
		scene.IDMatchmaking:  func() scene.Scene { return scene.NewMatchmaking(mgr) },
		scene.IDGame:         func() scene.Scene { return scene.NewGame(mgr) },
		scene.IDResult:       func() scene.Scene { return scene.NewResult(mgr, "", "", 0, 0, 0) },
		scene.IDProfile:      func() scene.Scene { return scene.NewProfile(mgr) },
		scene.IDProfileOther: func() scene.Scene { return scene.NewProfileOther(mgr, "") },
		scene.IDLeaderboard:  func() scene.Scene { return scene.NewLeaderboard(mgr) },
	}

	mgr = scene.NewManager(factories)
	return mgr
}
