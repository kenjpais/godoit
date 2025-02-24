package utils

import (
	"doit/internal/db"
)

type JobItem struct {
	Value    *db.Job
	Priority int
	Index    int
}

type PriorityQueue []*JobItem

func (pq PriorityQueue) Len() int { return len(pq) }

func (pq PriorityQueue) Less(i, j int) bool {
	return pq[i].Priority > pq[j].Priority
}

func (pq PriorityQueue) Swap(i, j int) {
	pq[i], pq[j] = pq[j], pq[i]
	pq[i].Index = i
	pq[j].Index = j
}

func (pq *PriorityQueue) Push(x interface{}) {
	n := len(*pq)
	JobItem := x.(*JobItem)
	JobItem.Index = n
	*pq = append(*pq, JobItem)
}

func (pq *PriorityQueue) Pop() interface{} {
	old := *pq
	n := len(old)
	JobItem := old[n-1]
	*pq = old[0 : n-1]
	return JobItem
}