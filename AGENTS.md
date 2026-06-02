# Connect 4 — Agent Context

## Project structure
```
connect-4/
├── AGENTS.md                     ← you are here
├── README.md
├── build.sh                      ./build.sh  (wasm + server, then DATABASE_URL=... JWT_SECRET=... ./server-bin)
├── Dockerfile                    multi-stage (wasm builder → server builder → alpine runtime)
├── client/
│   ├── main.go                   Ebitengine entry, 1024×768 window
│   ├── go.mod / go.sum
│   ├── assets/
│   │   ├── assets.go             go:embed MustLoadImage
│   │   └── images/               PNGs (backgrounds/, birds/, main/)
│   ├── net/
│   │   ├── http.go               browser fetch POST/GET
│   │   └── ws.go                 browser WebSocket wrapper
│   ├── scene/
│   │   ├── manager.go            Scene interface + Manager with navigation stack
│   │   ├── login.go              Login/Register — STUBBED
│   │   ├── menu.go               Main menu — STUBBED
│   │   ├── matchmaking.go        Queue/search — STUBBED
│   │   ├── game.go               Live board — STUBBED
│   │   ├── result.go             Post-game outcome — STUBBED
│   │   ├── profile.go            Own profile — STUBBED
│   │   ├── profile_other.go      Other user's profile — STUBBED
│   │   └── leaderboard.go        Leaderboard — STUBBED
│   ├── session/
│   │   └── session.go            Global auth State singleton
│   └── ui/                       Reusable UI components (buttons, inputs, scroll lists)
├── server/
│   ├── cmd/server/main.go        Entry point, routes, goose migrations
│   ├── go.mod / go.sum
│   ├── db/migrations/            goose SQL files
│   └── internal/
│       ├── auth/                 bcrypt, JWT, middleware
│       ├── db/                   pgx pool + queries
│       ├── elo/                  pure ELO calculation
│       ├── game/                 board, match, matchmaker
│       ├── handler/              HTTP + WS handlers
│       └── model/                shared types
├── static/
│   ├── index.html                iframe wrapper (1024×768)
│   ├── game.html                 WASM loader
│   ├── game.wasm                 (built, .gitignored)
│   └── wasm_exec.js              (copied from GOROOT, .gitignored)
└── k8s/                          namespace, postgres, deployment, secrets template
```

## Client architecture
- **Screen size**: 1024×768 (Ebitengine window + iframe wrapper)
- **Scene manager**: `scene.Manager` with a navigation stack. Every scene implements `Update() error` and `Draw(screen *ebiten.Image)`. Factories are called fresh on each `Navigate()` call.
- **Navigation**: `mgr.Navigate(id)`, `mgr.Back()`, `mgr.Reset()` (goes to login). `Back()` pops the stack — used for back buttons in profile/leaderboard/etc.
- **Session**: `session.Current` (singleton) holds `Token`, `Username`, `ELO`, `LoggedIn`.
- **Networking**: `net.Post(path, body, result)`, `net.Get(path, token, result)` use browser `fetch`. `net.NewWSConn()` opens `/ws` and returns channels for recv/done.

## Server architecture
- **Framework**: Go 1.22 `http.NewServeMux` with path params (`{username}`)
- **Database**: PostgreSQL via pgx pool, goose migrations
- **Auth**: bcrypt passwords, JWT (HS256, 72h expiry), Bearer token middleware
- **Game logic**: Pure Connect 4 rules (6×7 board, 4-direction win detection), ELO calculation
- **Matchmaker**: Single-slot queue — pairs two waiting players instantly, manages reconnect lookup
- **Grace period**: 30-second reconnect window when a player disconnects

### HTTP routes
| Method | Path | Description |
|--------|------|-------------|
| POST | `/register` | Create user → returns `{token, username, elo}` |
| POST | `/login` | Authenticate → returns `{token, username, elo}` |
| GET | `/leaderboard` | All users by ELO desc → `[{rank, username, elo, wins, losses, draws}]` |
| GET | `/profile/{username}` | Public profile → `{username, elo, wins, losses, draws, history}` |
| GET | `/ws` | WebSocket (requires JWT Bearer middleware) |

### WebSocket protocol

**Client → Server** (inbound):
| Type | Payload | When |
|------|---------|------|
| `auth` | `{token: "<JWT>"}` | First message on connect |
| `queue` | — | Join matchmaking queue |
| `cancel` | — | Leave queue |
| `place` | `{column: N}` | Drop token (0-6) |

**Server → Client** (outbound):
| Type | Payload | When |
|------|---------|------|
| `error` | `{message: "..."}` | Validation/state errors |
| `waiting` | — | Acknowledged queue join |
| `cancelled` | — | Acknowledged queue leave |
| `matched` | `{opponent_name, your_color, your_turn}` | Game starting |
| `state` | `{board: [[...]], turn: N}` | After every move |
| `opponent_disconnected` | `{seconds_remaining: N}` | Tick every second during grace |
| `opponent_reconnected` | — | Opponent rejoined |
| `result` | `{outcome, win_color, elo_before, elo_after, elo_delta}` | Game over |

### Database schema
- **users**: `id SERIAL PK`, `username TEXT UNIQUE`, `password_hash TEXT`, `elo INT DEFAULT 1000`, `created_at TIMESTAMPTZ`
- **matches**: `id SERIAL PK`, `player1_id INT FK`, `player2_id INT FK`, `winner_id INT FK (NULL=draw)`, `player1_elo_before INT`, `player2_elo_before INT`, `elo_delta INT`, `played_at TIMESTAMPTZ`

## Color scheme
```go
airForceBlue = color.RGBA{0x66, 0x89, 0xa1, 0xff}
frostedMint  = color.RGBA{0xd5, 0xf9, 0xde, 0xff}
deepWalnut   = color.RGBA{0x53, 0x3e, 0x2d, 0xff}
powderBlush  = color.RGBA{0xed, 0xb6, 0xa3, 0xff}
darkCyan     = color.RGBA{0x11, 0x9d, 0xa4, 0xff}
```

## Implementation order

- [x] 1. `client/assets/assets.go` — `MustLoadImage` with `//go:embed`
- [x] 2. `client/ui/button.go` + `client/ui/input.go` — reusable UI components
- [x] 3. `scene/login.go` — keyboard input, login/register toggle, HTTP calls, error display
- [ ] 4. `scene/menu.go` — 4 buttons: Find Match, Profile, Leaderboard, Sign Out
- [ ] 5. `scene/matchmaking.go` — WS connect, auth+queue, handle "matched", cancel
- [ ] 6. `scene/game.go` — board + token rendering, mouse column hover, WS state/result/disconnect handling
- [ ] 7. `scene/result.go` — outcome display (win/loss/draw + color + ELO delta), back to menu
- [ ] 8. `scene/profile.go` — HTTP GET /profile/{username}, stats, scrollable match history with clickable names
- [ ] 9. `scene/profile_other.go` — same as profile but for any user by URL
- [ ] 10. `scene/leaderboard.go` — HTTP GET /leaderboard, scrollable table with clickable usernames
- [ ] 11. Wire disconnect overlay in game.go (30s countdown, hide on reconnect)
- [ ] 12. Polish — button hover states, error toasts, loading spinners, keyboard shortcuts

## Build & View Instructions

**Every time you make progress, run these 4 steps to see it live:**

```bash
# Step 1: Build the WASM client
cd client && GOOS=js GOARCH=wasm go build -o ../static/game.wasm . && cd ..

# Step 2: Copy wasm_exec.js from Go's stdlib
# (one of these two paths will exist)
cp "$(go env GOROOT)/lib/wasm/wasm_exec.js" static/wasm_exec.js 2>/dev/null || \
cp "$(go env GOROOT)/misc/wasm/wasm_exec.js" static/wasm_exec.js

# Step 3: Build the server binary
cd server && go build -o ../server-bin ./cmd/server && cd ..

# Step 4: Run the server with your DB
DATABASE_URL=postgres://user:password@localhost:5432/connect4?sslmode=disable \
JWT_SECRET=your-secret-here \
./server-bin

# Step 5: Open http://localhost:8080 in your browser
```

Or use the convenience script (skips setting env vars):
```bash
./build.sh
# then set DATABASE_URL and JWT_SECRET manually when running server-bin
```

**Hot-reload shortcut** (after making code changes, just re-run steps 1, 3, and restart the server):
```bash
cd client && GOOS=js GOARCH=wasm go build -o ../static/game.wasm . && cd .. && \
cd server && go build -o ../server-bin ./cmd/server && cd .. && \
kill ./server-bin 2>/dev/null; \
DATABASE_URL=postgres://user:password@localhost:5432/connect4?sslmode=disable \
JWT_SECRET=your-secret-here \
./server-bin
```

> **Note**: You need a running PostgreSQL instance and a database named `connect4`. The server runs goose migrations automatically on startup.
