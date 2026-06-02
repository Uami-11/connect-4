// Package game contains all Connect 4 game logic.
// No database, no network — pure board state and rules.
package game

import "errors"

const (
	Rows = 6
	Cols = 7

	Empty  = 0
	Red    = 1 // Player 1
	Yellow = 2 // Player 2
)

// Board is the 6×7 Connect 4 grid.
// board[0] is the bottom row, board[5] is the top.
type Board [Rows][Cols]int

// Drop places a token for the given player in the given column.
// Returns the row it landed on, or an error if the column is full or invalid.
func (b *Board) Drop(col, player int) (row int, err error) {
	if col < 0 || col >= Cols {
		return 0, errors.New("column out of range")
	}
	for r := 0; r < Rows; r++ {
		if b[r][col] == Empty {
			b[r][col] = player
			return r, nil
		}
	}
	return 0, errors.New("column is full")
}

// CheckWin returns true if the given player has four in a row
// anywhere on the board.
func (b *Board) CheckWin(player int) bool {
	// Horizontal
	for r := 0; r < Rows; r++ {
		for c := 0; c <= Cols-4; c++ {
			if b[r][c] == player && b[r][c+1] == player &&
				b[r][c+2] == player && b[r][c+3] == player {
				return true
			}
		}
	}
	// Vertical
	for r := 0; r <= Rows-4; r++ {
		for c := 0; c < Cols; c++ {
			if b[r][c] == player && b[r+1][c] == player &&
				b[r+2][c] == player && b[r+3][c] == player {
				return true
			}
		}
	}
	// Diagonal ↗
	for r := 0; r <= Rows-4; r++ {
		for c := 0; c <= Cols-4; c++ {
			if b[r][c] == player && b[r+1][c+1] == player &&
				b[r+2][c+2] == player && b[r+3][c+3] == player {
				return true
			}
		}
	}
	// Diagonal ↘
	for r := 3; r < Rows; r++ {
		for c := 0; c <= Cols-4; c++ {
			if b[r][c] == player && b[r-1][c+1] == player &&
				b[r-2][c+2] == player && b[r-3][c+3] == player {
				return true
			}
		}
	}
	return false
}

// IsFull returns true when every cell is occupied (draw condition).
func (b *Board) IsFull() bool {
	for c := 0; c < Cols; c++ {
		if b[Rows-1][c] == Empty {
			return false
		}
	}
	return true
}
