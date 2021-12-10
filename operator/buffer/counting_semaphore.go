package buffer

import (
	"container/list"
	"context"
	"sync"
)

// CountingSemaphore is a classical counting semaphore; It holds a value that may be incremented and decremented.
// Waiting on the semaphore blocks until the value is greater than 0, and then decrements that count by one before returning.
// Incrementing the value will either increment the internal value, or release a waiting thread.
type CountingSemaphore struct {
	val      int64
	mux      *sync.Mutex
	waitList list.List
}

// NewCountingSemaphore returns
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

	next := rs.waitList.Front()
	if next == nil {
		// No reader to notify
		rs.val += 1
		return
	}

	notify := next.Value.(chan struct{})
	rs.waitList.Remove(next)
	close(notify)
}

// Acquire waits for a resource to be available, and decrements the amount of the resource available.
// It is possible for the resource to be acquired, even if the context is already cancelled.
func (rs *CountingSemaphore) Acquire(ctx context.Context) error {
	rs.mux.Lock()

	if rs.val > 0 {
		rs.val -= 1
		rs.mux.Unlock()
		return nil
	}

	signal := make(chan struct{})
	elem := rs.waitList.PushBack(signal)

	rs.mux.Unlock()

	select {
	case <-ctx.Done():
		err := ctx.Err()
		rs.mux.Lock()
		select {
		case <-signal:
			// We were already signalled, so we must ignore context cancellation
			// (pretend we didn't see it)
			err = nil
		default:
			rs.waitList.Remove(elem)
		}
		rs.mux.Unlock()
		return err
	case <-signal:
		return nil
	}
}
