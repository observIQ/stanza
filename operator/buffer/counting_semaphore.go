package buffer

import (
	"container/list"
	"context"
	"sync"
	"time"
)

// CountingSemaphore is a classical counting semaphore; It holds a value that may be incremented and decremented.
// Waiting on the semaphore blocks until the value is greater than 0, and then decrements that count by one before returning.
// Incrementing the value will either increment the internal value, or release a waiting thread.
type CountingSemaphore struct {
	val      int64
	mux      *sync.Mutex
	waitList list.List
}

type waitListItem struct {
	signal chan struct{}
	n      int64
}

// NewCountingSemaphore returns new counting semephore, with it's internal value set to initialVal
func NewCountingSemaphore(initialVal int64) *CountingSemaphore {
	return &CountingSemaphore{
		val: initialVal,
		mux: &sync.Mutex{},
	}
}

// Increment will attempt to wake a thread waiting on the ResourceSemaphore;
// If there is no such waiter, it will increment the amount of the resource available.
func (rs *CountingSemaphore) Increment() {
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
	close(item.signal)
}

// // Acquire waits for a resource to be available, and decrements the amount of the resource available.
// // It is possible for the resource to be acquired, even if the context is already cancelled.
// func (rs *CountingSemaphore) Acquire(ctx context.Context) error {
// 	rs.mux.Lock()

// 	if rs.val > 0 {
// 		rs.val -= 1
// 		rs.mux.Unlock()
// 		return nil
// 	}

// 	signal := make(chan struct{})
// 	elem := rs.waitList.PushBack(signal)

// 	rs.mux.Unlock()

// 	select {
// 	case <-ctx.Done():
// 		err := ctx.Err()
// 		rs.mux.Lock()
// 		select {
// 		case <-signal:
// 			// We were already signalled, so we must ignore context cancellation
// 			// (pretend we didn't see it)
// 			err = nil
// 		default:
// 			rs.waitList.Remove(elem)
// 		}
// 		rs.mux.Unlock()
// 		return err
// 	case <-signal:
// 		return nil
// 	}
// }

// // TryAcquire attempts to decrement the value; If it succeeds, returns true.
// func (rs *CountingSemaphore) TryAcquire() bool {
// 	rs.mux.Lock()
// 	defer rs.mux.Unlock()

// 	if rs.val > 0 {
// 		rs.val -= 1
// 		return true
// 	}

// 	return false
// }

// AcquireAtMost acquires at most n resource.
// If it cannot acquire at n resource, it will block until the context cancels, or a timeout occurs.
// If n resource cannot be acquired, then as much as possible will be acquired.
// Returns the amount of resource acquired.
func (cs *CountingSemaphore) AcquireAtMost(ctx context.Context, timeout time.Duration, n int64) int64 {
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

func (cs *CountingSemaphore) doAcquire(n int64, elem *list.Element) int64 {
	var amountToTake int64
	signal := elem.Value.(waitListItem).signal
	cs.mux.Lock()
	defer cs.mux.Unlock()

	select {
	case <-signal:
		// We were already signalled, so we must ignore context cancellation
		// (pretend we didn't see it)
		amountToTake = n
	default:
		if cs.val > n {
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
