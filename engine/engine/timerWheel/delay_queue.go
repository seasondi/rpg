package timerWheel

import (
	"container/heap"
)

type Item struct {
	Value    *Bucket
	Priority int64
	Index    int
}

type priorityQueue []*Item

func NewPriorityQueue(capacity int) priorityQueue {
	return make(priorityQueue, 0, capacity)
}

func (m priorityQueue) Len() int { return len(m) }

func (m priorityQueue) Less(i, j int) bool { return m[i].Priority < m[j].Priority }

func (m priorityQueue) Swap(i, j int) {
	m[i], m[j] = m[j], m[i]
	m[i].Index, m[j].Index = i, j
}

func (m *priorityQueue) Push(x interface{}) {
	n := len(*m)
	c := cap(*m)
	if n+1 > c {
		newQueue := make(priorityQueue, n, c*2)
		copy(newQueue, *m)
		*m = newQueue
	}
	*m = (*m)[0 : n+1]
	item := x.(*Item)
	item.Index = n
	(*m)[n] = item
}

func (m *priorityQueue) Pop() interface{} {
	n := len(*m)
	c := cap(*m)
	if n < (c/2) && c > 25 {
		newQueue := make(priorityQueue, n, c/2)
		copy(newQueue, *m)
		*m = newQueue
	}
	item := (*m)[n-1]
	item.Index = -1
	*m = (*m)[0 : n-1]
	return item
}

func (m *priorityQueue) PeekAndShift(max int64) *Item {
	if m.Len() == 0 {
		return nil
	}

	item := (*m)[0]
	if item.Priority > max {
		return nil
	}
	heap.Remove(m, 0)

	item.Value.SetIsInQueue(false)
	return item
}

type PriorityQueue struct {
	pq priorityQueue
}

func NewQueue(size int) *PriorityQueue {
	return &PriorityQueue{
		pq: NewPriorityQueue(size),
	}
}

func (m *PriorityQueue) Add(elem *Bucket, expiration int64) {
	elem.SetIsInQueue(true)
	item := &Item{Value: elem, Priority: expiration}
	heap.Push(&m.pq, item)
}

func (m *PriorityQueue) GetExpiredAndShift(expiration int64) []*Bucket {
	r := make([]*Bucket, 0, 0)
	for item := m.pq.PeekAndShift(expiration); item != nil; item = m.pq.PeekAndShift(expiration) {
		r = append(r, item.Value)
	}
	return r
}
