package taskheap

import (
	"github.com/shengmingzhu/timercron/timertask"
	"time"
)

type TaskHeap struct {
	Heap []*timertask.Task
	len  int
}

func New() *TaskHeap {
	h := &TaskHeap{
		Heap: make([]*timertask.Task, 0),
		len:  0,
	}
	return h
}

// Push a task into the heap
// O(log(N))
func (h *TaskHeap) Push(t *timertask.Task) {
	if t == nil || t.Disable {
		return
	}

	if h.len < len(h.Heap) {
		h.Heap[h.len] = t
	} else {
		h.Heap = append(h.Heap, t)
	}
	h.len++

	h.adjustLastElement()
}

// Pop returns a recently task.
// The disabled tasks will be removed in this function, when the tasks been adjusted at the bottom of the heap before we call Pop().
// O(log(N) + K), K is the number of disabled tasks at the bottom of the heap, usually K is very small.
func (h *TaskHeap) Pop() *timertask.Task {
	return h.popAndAdjust()
}

// Nearest returns the nearest task's nextExecTime.
// O(1)
func (h *TaskHeap) Nearest() time.Duration {
	if h.len <= 0 {
		return -1
	}
	return h.Heap[0].GetNextExecTime()
}

// adjustFirstElement pops up the minimum element, also it's the first one.
// then we move the last one to first, and adjust heap.
func (h *TaskHeap) popAndAdjust() *timertask.Task {
	if h == nil || h.len <= 0 {
		return nil
	}
	t := h.Heap[0]
	h.len--
	h.Heap[0] = h.Heap[h.len] // move the last element to first position
	if !h.Heap[0].CheckParameters() {
		// here we remove the disabled task
		// It may be because the task EndTime or ExpectTimes are up, or the user set the task.disable = true.
		_ = h.popAndAdjust()
	}
	h.adjustFirstElement() // then adjust the heap

	return t
}

// adjustLastElement adjusts the last element to the correct position
// it need be called when add()
func (h *TaskHeap) adjustLastElement() {
	for i := h.len - 1; i > 0; {
		if h.Heap[i].GetNextExecTime() >= h.Heap[((i+1)>>1)-1].GetNextExecTime() {
			break
		}
		h.Heap[i], h.Heap[((i+1)>>1)-1] = h.Heap[((i+1)>>1)-1], h.Heap[i]
		i = ((i + 1) >> 1) - 1
	}
}

// adjustFirstElement adjusts the first element to the correct position
// it need be called when pop()
func (h *TaskHeap) adjustFirstElement() {
	for i := 0; (i+1)<<1 <= h.len; {
		var minChild int
		if h.Heap[((i+1)<<1)-1].GetNextExecTime() < h.Heap[(i+1)<<1].GetNextExecTime() {
			minChild = ((i + 1) << 1) - 1
		} else {
			minChild = (i + 1) << 1
		}

		if h.Heap[i].GetNextExecTime() < h.Heap[minChild].GetNextExecTime() {
			break
		} else {
			h.Heap[i], h.Heap[minChild] = h.Heap[minChild], h.Heap[i]
			i = minChild
		}
	}
}
