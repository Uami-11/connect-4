package main

import (
	"context"
	"database/sql"
	"embed"
	"log"
	"net/http"
	"os"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"

	"connect4/server/internal/db"
	"connect4/server/internal/game"
	"connect4/server/internal/handler"
)

//go:embed db/migrations/*.sql
var migrations embed.FS

func main() {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		log.Fatal("DATABASE_URL environment variable is required")
	}
	jwtSecret := os.Getenv("JWT_SECRET")
	if jwtSecret == "" {
		log.Fatal("JWT_SECRET environment variable is required")
	}

	// Run goose migrations.
	sqlDB, err := sql.Open("pgx", dsn)
	if err != nil {
		log.Fatalf("opening sql db for migrations: %v", err)
	}
	goose.SetBaseFS(migrations)
	if err := goose.SetDialect("postgres"); err != nil {
		log.Fatalf("goose set dialect: %v", err)
	}
	if err := goose.Up(sqlDB, "db/migrations"); err != nil {
		log.Fatalf("running migrations: %v", err)
	}
	sqlDB.Close()

	// Open pgx pool.
	pool, err := db.Connect(context.Background(), dsn)
	if err != nil {
		log.Fatalf("connecting to database: %v", err)
	}
	defer pool.Close()

	queries := db.New(pool)
	mm := game.NewMatchmaker(queries)

	// Handlers.
	authHandler := handler.NewAuth(queries, jwtSecret)
	leaderboardHandler := handler.NewLeaderboard(queries)
	profileHandler := handler.NewProfile(queries)
	wsHandler := handler.NewWS(mm, jwtSecret)

	mux := http.NewServeMux()

	// API routes.
	mux.HandleFunc("POST /register", authHandler.Register)
	mux.HandleFunc("POST /login", authHandler.Login)
	mux.HandleFunc("GET /leaderboard", leaderboardHandler.Get)
	mux.HandleFunc("GET /profile/{username}", profileHandler.Get)
	mux.Handle("/ws", http.HandlerFunc(wsHandler.ServeHTTP))

	// Serve WASM client static files.
	mux.Handle("/", http.FileServer(http.Dir("./static")))

	log.Println("connect4 server listening on :8080")
	log.Fatal(http.ListenAndServe(":8080", mux))
}
