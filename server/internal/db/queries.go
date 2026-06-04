package db

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"connect4/server/internal/model"
)

// Queries wraps the connection pool and exposes typed query functions.
type Queries struct {
	pool *pgxpool.Pool
}

// New returns a Queries bound to the given pool.
func New(pool *pgxpool.Pool) *Queries {
	return &Queries{pool: pool}
}

// --- User queries ---

// CreateUser inserts a new user and returns their assigned ID.
func (q *Queries) CreateUser(ctx context.Context, username, passwordHash string) (int, error) {
	var id int
	err := q.pool.QueryRow(ctx,
		`INSERT INTO users (username, password_hash, elo)
		 VALUES ($1, $2, 600)
		 RETURNING id`,
		username, passwordHash,
	).Scan(&id)
	if err != nil {
		return 0, fmt.Errorf("creating user: %w", err)
	}
	return id, nil
}

// GetUserByUsername returns the full user row including password hash.
func (q *Queries) GetUserByUsername(ctx context.Context, username string) (*model.User, error) {
	u := &model.User{}
	err := q.pool.QueryRow(ctx,
		`SELECT id, username, password_hash, elo, created_at, last_active
		 FROM users WHERE username = $1`,
		username,
	).Scan(&u.ID, &u.Username, &u.PasswordHash, &u.ELO, &u.CreatedAt, &u.LastActiveAt)
	if err != nil {
		return nil, fmt.Errorf("getting user %q: %w", username, err)
	}
	return u, nil
}

// GetELOByID returns the current ELO for a user.
func (q *Queries) GetELOByID(ctx context.Context, userID int) (int, error) {
	var elo int
	err := q.pool.QueryRow(ctx, `SELECT elo FROM users WHERE id = $1`, userID).Scan(&elo)
	if err != nil {
		return 0, fmt.Errorf("getting elo for user %d: %w", userID, err)
	}
	return elo, nil
}

// UpdateELO sets a player's ELO to the new value.
func (q *Queries) UpdateELO(ctx context.Context, userID, newELO int) error {
	_, err := q.pool.Exec(ctx,
		`UPDATE users SET elo = $1 WHERE id = $2`,
		newELO, userID,
	)
	if err != nil {
		return fmt.Errorf("updating elo for user %d: %w", userID, err)
	}
	return nil
}

// UpdateLastActive sets a player's last_active timestamp to now.
func (q *Queries) UpdateLastActive(ctx context.Context, userID int) error {
	_, err := q.pool.Exec(ctx,
		`UPDATE users SET last_active = NOW() WHERE id = $1`,
		userID,
	)
	if err != nil {
		return fmt.Errorf("updating last_active for user %d: %w", userID, err)
	}
	return nil
}

// --- Match queries ---

// CreateMatch inserts a completed match record.
// winnerID is nil for a draw.
func (q *Queries) CreateMatch(ctx context.Context, m *model.Match) error {
	_, err := q.pool.Exec(ctx,
		`INSERT INTO matches
		   (player1_id, player2_id, winner_id, player1_elo_before, player2_elo_before, elo_delta, played_at)
		 VALUES ($1, $2, $3, $4, $5, $6, NOW())`,
		m.Player1ID, m.Player2ID, m.WinnerID,
		m.Player1ELOBefore, m.Player2ELOBefore, m.ELODelta,
	)
	if err != nil {
		return fmt.Errorf("inserting match: %w", err)
	}
	return nil
}

// --- Leaderboard query ---

// GetLeaderboard returns all users ranked by ELO descending,
// with win/loss/draw counts computed from the matches table.
func (q *Queries) GetLeaderboard(ctx context.Context) ([]model.LeaderboardEntry, error) {
	rows, err := q.pool.Query(ctx, `
		SELECT
			DENSE_RANK() OVER (ORDER BY u.elo DESC) AS rank,
			u.username,
			u.elo,
			COUNT(CASE WHEN m.winner_id = u.id THEN 1 END)                             AS wins,
			COUNT(CASE WHEN (m.player1_id = u.id OR m.player2_id = u.id)
			            AND m.winner_id IS NOT NULL
			            AND m.winner_id != u.id THEN 1 END)                            AS losses,
			COUNT(CASE WHEN (m.player1_id = u.id OR m.player2_id = u.id)
			            AND m.winner_id IS NULL THEN 1 END)                            AS draws
		FROM users u
		LEFT JOIN matches m ON m.player1_id = u.id OR m.player2_id = u.id
		GROUP BY u.id, u.username, u.elo
		ORDER BY u.elo DESC, u.username ASC
	`)
	if err != nil {
		return nil, fmt.Errorf("querying leaderboard: %w", err)
	}
	defer rows.Close()

	var entries []model.LeaderboardEntry
	for rows.Next() {
		var e model.LeaderboardEntry
		if err := rows.Scan(&e.Rank, &e.Username, &e.ELO, &e.Wins, &e.Losses, &e.Draws); err != nil {
			return nil, fmt.Errorf("scanning leaderboard row: %w", err)
		}
		entries = append(entries, e)
	}
	return entries, rows.Err()
}

// --- Profile query ---

// GetPublicProfile returns a player's profile and last 20 matches.
func (q *Queries) GetPublicProfile(ctx context.Context, username string) (*model.PublicProfile, error) {
	// Get user row first.
	u, err := q.GetUserByUsername(ctx, username)
	if err != nil {
		return nil, err
	}

	// Aggregate win/loss/draw counts and profile metadata.
	p := &model.PublicProfile{
		Username:     u.Username,
		ELO:          u.ELO,
		CreatedAt:    u.CreatedAt,
		LastActiveAt: u.LastActiveAt,
	}
	if u.LastActiveAt != nil && time.Since(*u.LastActiveAt).Seconds() < 60 {
		p.Online = true
	}
	err = q.pool.QueryRow(ctx, `
		SELECT
			COUNT(CASE WHEN winner_id = $1 THEN 1 END)                                AS wins,
			COUNT(CASE WHEN (player1_id = $1 OR player2_id = $1)
			           AND winner_id IS NOT NULL AND winner_id != $1 THEN 1 END)      AS losses,
			COUNT(CASE WHEN (player1_id = $1 OR player2_id = $1)
			           AND winner_id IS NULL THEN 1 END)                              AS draws
		FROM matches
		WHERE player1_id = $1 OR player2_id = $1
	`, u.ID).Scan(&p.Wins, &p.Losses, &p.Draws)
	if err != nil {
		return nil, fmt.Errorf("aggregating profile stats: %w", err)
	}

	// Last 20 matches with opponent name and ELO delta.
	rows, err := q.pool.Query(ctx, `
		SELECT
			m.id,
			opp.username                                        AS opponent_name,
			CASE
				WHEN m.winner_id = $1 THEN 'win'
				WHEN m.winner_id IS NULL THEN 'draw'
				ELSE 'loss'
			END                                                 AS outcome,
			CASE WHEN m.player1_id = $1
				THEN m.player1_elo_before
				ELSE m.player2_elo_before
			END                                                 AS elo_before,
			CASE
				WHEN m.winner_id = $1
					THEN CASE WHEN m.player1_id = $1
						THEN m.player1_elo_before + m.elo_delta
						ELSE m.player2_elo_before + m.elo_delta END
				WHEN m.winner_id IS NULL
					THEN CASE WHEN m.player1_id = $1
						THEN m.player1_elo_before
						ELSE m.player2_elo_before END
				ELSE CASE WHEN m.player1_id = $1
					THEN m.player1_elo_before - m.elo_delta
					ELSE m.player2_elo_before - m.elo_delta END
			END                                                 AS elo_after,
			m.elo_delta,
			m.played_at
		FROM matches m
		JOIN users opp ON opp.id = CASE
			WHEN m.player1_id = $1 THEN m.player2_id
			ELSE m.player1_id
		END
		WHERE m.player1_id = $1 OR m.player2_id = $1
		ORDER BY m.played_at DESC
		LIMIT 20
	`, u.ID)
	if err != nil {
		return nil, fmt.Errorf("querying match history: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var e model.MatchHistoryEntry
		if err := rows.Scan(
			&e.MatchID, &e.OpponentName, &e.Outcome,
			&e.ELOBefore, &e.ELOAfter, &e.ELODelta, &e.PlayedAt,
		); err != nil {
			return nil, fmt.Errorf("scanning history row: %w", err)
		}
		p.History = append(p.History, e)
	}
	return p, rows.Err()
}
