package client

import (
	"github.com/fatih/color"
)

var systempq = color.New(color.FgHiMagenta).Add(color.BgBlack)

type PriorityQueue []PqEntry

func (pq PriorityQueue) Len() int { return len(pq) }

func (pq PriorityQueue) Less(i, j int) bool {
	// We want Pop to give us the highest, not lowest, priority so we use greater than here.
	return pq[i].timestamp < pq[j].timestamp
}

func (pq PriorityQueue) Swap(i, j int) {
	pq[i], pq[j] = pq[j], pq[i]
}

func (pq *PriorityQueue) Push(x any) {
	item := x.(PqEntry)
	systempq.Println(item, "is being added to the PQ")
	*pq = append(*pq, item)
}

func (pq *PriorityQueue) Pop() any {
	old := *pq
	n := len(old)
	item := old[n-1]
	old[n-1] = PqEntry{}  // avoid memory leak
	// item.Id = -1 // for safety
	systempq.Println(item, "has been removed from the PQ")
	*pq = old[0 : n-1]
	return item
}

// update modifies the priority and value of an Item in the queue.
// func (pq *PriorityQueue) update(item *Client, value string, local int) {
// 	heap.Fix(pq, item.Id)
// }
