package hn

import (
	"cmp"
	"fmt"
	"slices"
)

// Sortable defines methods for comparing baseItem
// and other structs (e.g., Comment, Story, etc.).
type Sortable interface {
	getID() uint
	getBy() string
	getScore() int
	getTime() Timestamp
	getType() string
}

// Order represents the sorting order: ascending or descending.
type Order int

const (
	Ascending Order = iota
	Descending
)

// Sort sorts the items using the specified sorting function.
func Sort[S Sortable](items []S, sort func(a, b S) int) {
	slices.SortStableFunc(items, sort)
}

// SortID sorts the items by ID according to the specified order.
func SortID[S Sortable](items []S, order Order) {
	switch order {
	case Ascending:
		Sort(items, func(a, b S) int {
			return cmp.Compare(a.getID(), b.getID())
		})
	case Descending:
		Sort(items, func(a, b S) int {
			return cmp.Compare(b.getID(), a.getID())
		})
	default:
		fmt.Printf("Invalid sort order: %v", order)
	}
}

// SortScore sorts the items by score according to the specified order.
func SortScore[S Sortable](items []S, order Order) {
	switch order {
	case Ascending:
		Sort(items, func(a, b S) int {
			return cmp.Compare(a.getScore(), b.getScore())
		})
	case Descending:
		Sort(items, func(a, b S) int {
			return cmp.Compare(b.getScore(), a.getScore())
		})
	default:
		fmt.Printf("Invalid sort order: %v", order)
	}
}

// SortTime sorts the items by creation time according to the specified order.
func SortTime[S Sortable](items []S, order Order) {
	switch order {
	case Ascending:
		Sort(items, func(a, b S) int {
			return cmp.Compare(a.getTime().UnixNano(), b.getTime().UnixNano())
		})
	case Descending:
		Sort(items, func(a, b S) int {
			return cmp.Compare(b.getTime().UnixNano(), a.getTime().UnixNano())
		})
	default:
		fmt.Printf("Invalid sort order: %v", order)
	}
}

// SortType sorts the items by type according to the specified order.
func SortType[S Sortable](items []S, order Order) {
	switch order {
	case Ascending:
		Sort(items, func(a, b S) int {
			return cmp.Compare(a.getType(), b.getType())
		})
	case Descending:
		Sort(items, func(a, b S) int {
			return cmp.Compare(b.getType(), a.getType())
		})
	default:
		fmt.Printf("Invalid sort order: %v", order)
	}
}
