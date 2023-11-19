package client

import (
	"container/heap"
	"sync"
)

// PriorityQueue implements a minimum priority queue for PQEntry.
type PriorityQueue struct {
	entries []*PQEntry
	mu      sync.Mutex
}

func (pq *PriorityQueue) Len() int {
	return len(pq.entries)
}

// Less defines the ordering of elements in the priority queue.
// It orders based on the lowest timestamp value. In case of a tie,
// it orders based on the ID.
func (pq *PriorityQueue) Less(i, j int) bool {
	if pq.entries[i].Timestamp == pq.entries[j].Timestamp {
		return pq.entries[i].ID < pq.entries[j].ID
	}
	return pq.entries[i].Timestamp < pq.entries[j].Timestamp
}

// Swap swaps the elements with indexes i and j.
func (pq *PriorityQueue) Swap(i, j int) { pq.entries[i], pq.entries[j] = pq.entries[j], pq.entries[i] }

// Push adds an element to the priority queue.
func (pq *PriorityQueue) Push(x interface{}) {
	entry := x.(*PQEntry)

	pq.entries = append(pq.entries, entry)
}

// Pop removes and returns the minimum element from the priority queue.
func (pq *PriorityQueue) Pop() interface{} {

	old := pq.entries
	n := len(old)
	entry := old[n-1]
	pq.entries = old[0 : n-1]
	return entry
}

// Remove removes an element from the priority queue based on ID.
// It returns true if the element is removed, or false if the ID doesn't exist.
func (pq *PriorityQueue) Remove(ID int) bool {
	index := -1
	// Find the index of the element with the given ID.
	for i, entry := range pq.entries {
		if entry.ID == ID {
			index = i
			break
		}
	}

	// If the element with the given ID is not found, return false.
	if index == -1 {
		return false
	}

	// If the element to be removed is not the last element,
	// swap it with the last element and heapify the priority queue.
	if index != len(pq.entries)-1 {
		pq.Swap(index, len(pq.entries)-1)
		pq.entries = pq.entries[:len(pq.entries)-1]
		heap.Fix(pq, index)
	} else {
		// Remove the last element.
		pq.entries = pq.entries[:len(pq.entries)-1]
	}

	return true
}

func (pq *PriorityQueue) ContainsTimestamp(timestamp int) bool {
	for _, entry := range pq.entries {
		if entry.Timestamp == timestamp {
			return true
		}
	}
	return false
}

func (pq *PriorityQueue) GetEntries() []*PQEntry {
	// Create a copy of the entries to avoid race conditions.
	entriesCopy := make([]*PQEntry, len(pq.entries))
	copy(entriesCopy, pq.entries)

	return entriesCopy
}