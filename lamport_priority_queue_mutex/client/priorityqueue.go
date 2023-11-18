package client

import (

)



// PriorityQueue implements a minimum priority queue for PQEntry.
type PriorityQueue []*PQEntry

// Len returns the number of elements in the priority queue.
func (pq PriorityQueue) Len() int { return len(pq) }

// Less defines the ordering of elements in the priority queue.
// It orders based on the lowest timestamp value. In case of a tie,
// it orders based on the ID.
func (pq PriorityQueue) Less(i, j int) bool {
	if pq[i].Timestamp == pq[j].Timestamp {
		return pq[i].ID < pq[j].ID
	}
	return pq[i].Timestamp < pq[j].Timestamp
}

// Swap swaps the elements with indexes i and j.
func (pq PriorityQueue) Swap(i, j int) { pq[i], pq[j] = pq[j], pq[i] }

// Push adds an element to the priority queue.
func (pq *PriorityQueue) Push(x interface{}) {
	entry := x.(*PQEntry)
	*pq = append(*pq, entry)
}

// Pop removes and returns the minimum element from the priority queue.
func (pq *PriorityQueue) Pop() interface{} {
	old := *pq
	n := len(old)
	entry := old[n-1]
	*pq = old[0 : n-1]
	return entry
}