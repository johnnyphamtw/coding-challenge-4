package data

import (
	"sort"
	"time"
)

// Board is a dated entry
type Board[T any] struct {
	Date time.Time
	Data T
}

// BoardLookup provides "latest on or before date" lookups
type BoardLookup[T any] struct {
	boards []Board[T] // sorted ascending by date
}

// NewBoardLookup creates a lookup from boards, sorting them by date
func NewBoardLookup[T any](boards []Board[T]) *BoardLookup[T] {
	sort.Slice(boards, func(i, j int) bool {
		return boards[i].Date.Before(boards[j].Date)
	})
	return &BoardLookup[T]{boards: boards}
}

// Lookup returns the data from the latest board on or before the given date
// Returns nil if no board exists on or before the date
func (bl *BoardLookup[T]) Lookup(date time.Time) *T {
	// Find the rightmost board with Date <= date
	idx := sort.Search(len(bl.boards), func(i int) bool {
		return bl.boards[i].Date.After(date)
	})
	// idx is the first board AFTER date, so idx-1 is the latest on or before
	if idx == 0 {
		return nil
	}
	return &bl.boards[idx-1].Data
}

// FirstDate returns the earliest board date, or zero time if empty
func (bl *BoardLookup[T]) FirstDate() time.Time {
	if len(bl.boards) == 0 {
		return time.Time{}
	}
	return bl.boards[0].Date
}
