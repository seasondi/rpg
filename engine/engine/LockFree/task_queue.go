package LockFree

import (
	"sync/atomic"
	"unsafe"
)

type ITaskHandler interface {
	HandleTask() error
}

type node struct {
	value ITaskHandler
	next  unsafe.Pointer
}

type TaskQueue struct {
	count      int32
	head, tail unsafe.Pointer
}

func NewTaskQueue() *TaskQueue {
	dummy := unsafe.Pointer(&node{})
	return &TaskQueue{
		head: dummy,
		tail: dummy,
	}
}

func (m *TaskQueue) Len() int {
	return int(atomic.LoadInt32(&m.count))
}

func (m *TaskQueue) Enqueue(task ITaskHandler) {
	newNode := unsafe.Pointer(&node{value: task})
	for {
		tail := atomic.LoadPointer(&m.tail)
		tailNode := (*node)(tail)
		next := atomic.LoadPointer(&tailNode.next)
		if next == nil {
			if atomic.CompareAndSwapPointer(&tailNode.next, nil, newNode) {
				atomic.CompareAndSwapPointer(&m.tail, tail, newNode)
				atomic.AddInt32(&m.count, 1)
				return
			}
		} else {
			atomic.CompareAndSwapPointer(&m.tail, tail, next)
		}
	}
}

func (m *TaskQueue) Dequeue() ITaskHandler {
	for {
		head := atomic.LoadPointer(&m.head)
		tail := atomic.LoadPointer(&m.tail)

		headNode := (*node)(head)
		next := atomic.LoadPointer(&headNode.next)

		if head == tail {
			if next == nil {
				return nil
			}
			atomic.CompareAndSwapPointer(&m.tail, tail, next)
		} else {
			value := (*node)(next).value
			if atomic.CompareAndSwapPointer(&m.head, head, next) {
				atomic.AddInt32(&m.count, -1)
				return value
			}
		}
	}
}
