//go:build js && wasm

package scene

import "github.com/hajimehoshi/ebiten/v2"

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
}

// NewManager creates a Manager with the given scene factories.
// Start() must be called once before any Update/Draw calls.
func NewManager(factories map[ID]func() Scene) *Manager {
	return &Manager{
		scenes: factories,
		stack:  []ID{IDLogin},
	}
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
	m.stack = []ID{IDLogin}
	m.current = m.scenes[IDLogin]()
}

// Update advances the current scene.
func (m *Manager) Update() error {
	return m.current.Update()
}

// Draw renders the current scene.
func (m *Manager) Draw(screen *ebiten.Image) {
	m.current.Draw(screen)
}
