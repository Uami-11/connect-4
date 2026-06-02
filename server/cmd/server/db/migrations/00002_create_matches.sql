-- +goose Up
CREATE TABLE matches (
    id                 SERIAL PRIMARY KEY,
    player1_id         INTEGER NOT NULL REFERENCES users(id),
    player2_id         INTEGER NOT NULL REFERENCES users(id),
    winner_id          INTEGER REFERENCES users(id),   -- NULL = draw
    player1_elo_before INTEGER NOT NULL,
    player2_elo_before INTEGER NOT NULL,
    elo_delta          INTEGER NOT NULL,
    played_at          TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_matches_player1 ON matches(player1_id);
CREATE INDEX idx_matches_player2 ON matches(player2_id);

-- +goose Down
DROP TABLE matches;
