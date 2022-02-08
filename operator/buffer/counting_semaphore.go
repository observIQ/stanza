package buffer

import (
	"container/list"
	"context"
	"sync"
	"time"
)

// GreedyCountingSemaphore is a classical counting semaphore, where waiting threads will greedily acquire up to n of the internal value.
// This code is based off the WeightedSemaphore implementation (https://cs.opensource.google/go/x/sync/+/036812b2:semaphore/semaphore.go)
type GreedyCountingSemaphore struct {
	val      int64
	mux      *sync.Mutex
	waitList list.List
}

type waitListItem struct {
	signal chan struct{}
	n      int64
}

// NewGreedyCountingSemaphore returns new counting semephore, with it's internal value set to initialVal
func NewGreedyCountingSemaphore(initialVal int64) *GreedyCountingSemaphore {
	return &GreedyCountingSemaphore{
		val: initialVal,
		mux: &sync.Mutex{},
	}
}

// Increment will increment the internal value of the semephore.
// If the first waiting thread can be released (acquire all n of its requested resource), then it will be released,
// and the internal value will be decremented accordingly.
func (rs *GreedyCountingSemaphore) Increment() {
	rs.mux.Lock()
	defer rs.mux.Unlock()

	rs.val += 1

	next := rs.waitList.Front()
	if next == nil {
		// No reader to notify
		return
	}

	item := next.Value.(waitListItem)

	if item.n > rs.val {
		// Cannot wake up waiting thread yet, we haven't hit the amount they are requesting
		return
	}

	rs.val -= item.n
	rs.waitList.Remove(next)
	// signal the waiting thread that they are clear to take n resource(s)
	close(item.signal)
}

// AcquireAtMost acquires at most n resource.
// If it cannot acquire n resource, it will block until the context cancels, or a timeout occurs.
// If n resource cannot be acquired by context cancellation or timeout, then as much resource as possible will be acquired.
// Returns the amount of resource acquired.
func (cs *GreedyCountingSemaphore) AcquireAtMost(ctx context.Context, timeout time.Duration, n int64) int64 {
	cs.mux.Lock()

	if cs.val >= n {
		cs.val -= n
		cs.mux.Unlock()
		return n
	}

	signal := make(chan struct{})
	elem := cs.waitList.PushBack(waitListItem{
		signal: signal,
		n:      n,
	})

	cs.mux.Unlock()

	timer := time.NewTimer(timeout)
	defer timer.Stop()

	select {
	case <-ctx.Done():
		return cs.doAcquire(n, elem)
	case <-timer.C:
		return cs.doAcquire(n, elem)
	case <-signal:
		// The list item has already been removed by the signaller;
		// we can just return n here.
		return n
	}
}

func (cs *GreedyCountingSemaphore) doAcquire(n int64, elem *list.Element) int64 {
	var amountToTake int64
	signal := elem.Value.(waitListItem).signal
	cs.mux.Lock()
	defer cs.mux.Unlock()

	select {
	case <-signal:
		// We were already signalled, so n resource was already allocated to this thread
		amountToTake = n
	default:
		if cs.val >= n {
			amountToTake = n
			cs.val -= n
		} else {
			amountToTake = cs.val
			cs.val = 0
		}
		cs.waitList.Remove(elem)
	}

	return amountToTake
}
