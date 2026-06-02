# Connect 4 Online

A real-time multiplayer Connect 4 game built with Go and Ebitengine (WASM client), featuring ELO rankings, match history, a global leaderboard, and a 30-second reconnect grace period.

## Tech Stack

- **Server**: Go 1.22, `net/http` (Go 1.22 ServeMux with path params), `gorilla/websocket`, `pgx` (PostgreSQL), `goose` (migrations), `golang-jwt`, `bcrypt`
- **Client**: Go compiled to WASM via `GOOS=js GOARCH=wasm`, Ebitengine v2 for 2D rendering
- **Database**: PostgreSQL 16
- **Infrastructure**: Docker multi-stage build, Kubernetes manifests (namespace, Postgres, deployment with ingress)

## Features

- **Account system**: Register and login with username/password
- **Matchmaking**: Queue up and get paired with the next available player
- **Live gameplay**: Turn-based Connect 4 on a 6×7 board
- **ELO ranking**: Ratings change after every match (K=32)
- **Reconnect grace period**: 30 seconds to rejoin after a disconnect
- **Match history**: View your past 20 matches with ELO delta per game
- **Leaderboard**: Global rankings sorted by ELO descending
- **Public profiles**: Click any username to see their stats and history

## Quick Start

### Prerequisites

- Go 1.22+
- PostgreSQL 16+
- Git

### 1. Clone and prepare

```bash
git clone <repo-url> connect4
cd connect4
```

### 2. Set up the database

```bash
createdb connect4
# or via psql:
# CREATE DATABASE connect4;
```

### 3. Build the WASM client

```bash
cd client
GOOS=js GOARCH=wasm go build -o ../static/game.wasm .
cd ..
```

### 4. Copy wasm_exec.js from Go's stdlib

```bash
cp "$(go env GOROOT)/lib/wasm/wasm_exec.js" static/wasm_exec.js 2>/dev/null || \
cp "$(go env GOROOT)/misc/wasm/wasm_exec.js" static/wasm_exec.js
```

### 5. Build and run the server

```bash
cd server
go build -o ../server-bin ./cmd/server
cd ..

DATABASE_URL=postgres://user:password@localhost:5432/connect4?sslmode=disable \
JWT_SECRET=change-me-to-a-random-string \
./server-bin
```

The server applies database migrations automatically on startup.

### 6. Open in browser

Navigate to [http://localhost:8080](http://localhost:8080)

### Convenience script

```bash
./build.sh   # builds WASM + server, copies wasm_exec.js
# then set DATABASE_URL and JWT_SECRET, run server-bin
```

## Architecture

```
┌─────────────┐     REST (login, register,     ┌──────────────┐     ┌────────────┐
│  Browser    │     leaderboard, profile)       │  Go Server   │────▶│ PostgreSQL │
│  (WASM +    │────────────────────────────────▶│  :8080       │     └────────────┘
│   Ebitengine)│                                │              │
│             │     WebSocket (matchmaking,     │  Handlers:   │
│  8 scenes:  │     game moves, reconnect)      │  Auth, WS,   │
│  login,     │◀────────────────────────────────│  Leaderboard,│
│  menu,      │                                │  Profile     │
│  matchmake, │                                │              │
│  game,      │     ┌──────────────────┐       │  Game engine:│
│  result,    │     │ Matchmaker (queue)│       │  board, match│
│  profile,   │     │ Match (game loop) │       │  ELO, auth   │
│  leaderboard│     │ Reconnect (30s)   │       │  migrations  │
└─────────────┘     └──────────────────┘       └──────────────┘
```

## API Reference

### HTTP Endpoints

| Method | Path | Auth | Request Body | Response |
|--------|------|------|-------------|----------|
| POST | `/register` | — | `{"username":"...","password":"..."}` | `{"token":"...","username":"...","elo":1000}` |
| POST | `/login` | — | `{"username":"...","password":"..."}` | `{"token":"...","username":"...","elo":...}` |
| GET | `/leaderboard` | — | — | `[{"rank":1,"username":"...","elo":...,"wins":...,"losses":...,"draws":...}]` |
| GET | `/profile/{username}` | — | — | `{"username":"...","elo":...,"wins":...,"losses":...,"draws":...,"history":[...]}` |
| GET | `/ws` | Bearer JWT | WebSocket upgrade | WebSocket |

### WebSocket Messages

See [AGENTS.md](./AGENTS.md) for the full message protocol.

## Project Structure

```
connect-4/
├── client/          Ebitengine WASM client
│   ├── main.go      Entry point, 1024×768
│   ├── assets/      Embedded images
│   ├── net/         HTTP + WebSocket helpers
│   ├── scene/       All 8 game screens
│   ├── session/     Auth state
│   └── ui/          Reusable components
├── server/          Go HTTP server
│   ├── cmd/server/  Entry point
│   ├── internal/    Auth, DB, ELO, game, handlers, models
│   └── db/migrations/ SQL migrations
├── static/          WASM loader + iframe wrapper
└── k8s/             Deployment manifests
```

## Deployment

The project includes Kubernetes manifests under `k8s/`. See `k8s/secrets.template.yaml` for required secrets.

The Dockerfile produces a minimal alpine image with the server binary and static assets wired together.

## License

MIT — see [LICENSE](./LICENSE). Copyright 2026 Nirwan Maharjan.
