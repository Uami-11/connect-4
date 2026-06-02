// Package elo implements the ELO rating system.
// All functions are pure — no side effects, no database calls.
package elo

import "math"

const (
	// K is the maximum ELO change per game.
	// 32 is standard for most platforms at this level.
	K = 32
)

// Outcome constants passed to Calculate.
const (
	Win  = 1.0
	Draw = 0.5
	Loss = 0.0
)

// expectedScore returns the expected score for player A against player B.
// This is the core ELO probability formula.
func expectedScore(ratingA, ratingB int) float64 {
	return 1.0 / (1.0 + math.Pow(10, float64(ratingB-ratingA)/400.0))
}

// Calculate returns the new ratings for both players after a game.
//
// outcome is from the perspective of player 1:
//   - Win  (1.0) = player 1 won
//   - Draw (0.5) = draw
//   - Loss (0.0) = player 1 lost
//
// Returns (newRating1, newRating2, delta) where delta is the absolute
// ELO points that changed hands (always positive).
func Calculate(rating1, rating2 int, outcome float64) (newRating1, newRating2, delta int) {
	e1 := expectedScore(rating1, rating2)
	e2 := expectedScore(rating2, rating1)

	r1 := float64(rating1) + K*(outcome-e1)
	r2 := float64(rating2) + K*((1-outcome)-e2)

	new1 := int(math.Round(r1))
	new2 := int(math.Round(r2))

	// Delta is always positive — callers decide who gained and who lost.
	d := new1 - rating1
	if d < 0 {
		d = -d
	}

	return new1, new2, d
}
