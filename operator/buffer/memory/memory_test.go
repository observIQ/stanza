package memory

import (
	"context"
	"math/rand"
	"strconv"
	"sync"
	"time"

	"testing"

	"github.com/observiq/stanza/entry"
	"github.com/stretchr/testify/require"
)

func intEntry(i int) *entry.Entry {
	e := entry.New()
	e.Timestamp = time.Date(2006, 01, 02, 03, 04, 05, 06, time.UTC)
	e.Record = float64(i)
	return e
}

func writeN(t testing.TB, buffer *MemoryBuffer, n, start int) {
	ctx := context.Background()
	for i := start; i < n+start; i++ {
		err := buffer.Add(ctx, intEntry(i))
		require.NoError(t, err)
	}
}

func readN(t testing.TB, buffer *MemoryBuffer, n, start int) func() {
	entries := make([]*entry.Entry, n)
	f, readCount, err := buffer.Read(entries)
	require.NoError(t, err)
	require.Equal(t, n, readCount)
	for i := 0; i < n; i++ {
		require.Equal(t, intEntry(start+i), entries[i])
	}
	return f
}

func readWaitN(t testing.TB, buffer *MemoryBuffer, n, start int) func() {
	entries := make([]*entry.Entry, n)
	f, readCount, err := buffer.ReadWait(entries, time.After(time.Minute))
	require.NoError(t, err)
	require.Equal(t, n, readCount)
	for i := 0; i < n; i++ {
		require.Equal(t, intEntry(start+i), entries[i])
	}
	return f
}

func uncheckedReadN(t testing.TB, buffer *MemoryBuffer, n int) func() {
	entries := make([]*entry.Entry, n)
	f, readCount, _ := buffer.Read(entries)
	require.Equal(t, n, readCount)
	return f
}

func flushN(t testing.TB, buffer *MemoryBuffer, n, start int) {
	f := readN(t, buffer, n, start)
	f()
}

func uncheckedFlushN(t testing.TB, buffer *MemoryBuffer, n int) {
	f := uncheckedReadN(t, buffer, n)
	f()
}

func TestMemoryBuffer(t *testing.T) {
	t.Run("Simple", func(t *testing.T) {
		t.Parallel()
		b := NewMemoryBuffer(1 << 10)
		writeN(t, b, 1, 0)
		readN(t, b, 1, 0)
	})

	t.Run("Write2Read1Read1", func(t *testing.T) {
		t.Parallel()
		b := NewMemoryBuffer(1 << 10)
		writeN(t, b, 2, 0)
		readN(t, b, 1, 0)
		readN(t, b, 1, 1)
	})

	t.Run("Write20Read10Read10", func(t *testing.T) {
		t.Parallel()
		b := NewMemoryBuffer(1 << 10)
		writeN(t, b, 20, 0)
		readN(t, b, 10, 0)
		readN(t, b, 10, 10)
	})

	t.Run("SingleReadWaitMultipleWrites", func(t *testing.T) {
		t.Parallel()
		b := NewMemoryBuffer(1 << 10)
		writeN(t, b, 10, 0)
		readyDone := make(chan struct{})
		go func() {
			readyDone <- struct{}{}
			readWaitN(t, b, 20, 0)
			readyDone <- struct{}{}
		}()
		<-readyDone
		time.Sleep(100 * time.Millisecond)
		writeN(t, b, 10, 10)
		<-readyDone
	})

	t.Run("ReadWaitOnlyWaitForPartialWrite", func(t *testing.T) {
		t.Parallel()
		b := NewMemoryBuffer(1 << 10)
		writeN(t, b, 10, 0)
		readyDone := make(chan struct{})
		go func() {
			readyDone <- struct{}{}
			readWaitN(t, b, 15, 0)
			readyDone <- struct{}{}
		}()
		<-readyDone
		writeN(t, b, 10, 10)
		<-readyDone
		readN(t, b, 5, 15)
	})

	t.Run("Write10Read10Read0", func(t *testing.T) {
		t.Parallel()
		b := NewMemoryBuffer(1 << 10)
		writeN(t, b, 10, 0)
		readN(t, b, 10, 0)
		dst := make([]*entry.Entry, 10)
		_, n, err := b.Read(dst)
		require.NoError(t, err)
		require.Equal(t, 0, n)
	})

	t.Run("Write20Read10Read10Unfull", func(t *testing.T) {
		t.Parallel()
		b := NewMemoryBuffer(1 << 10)
		writeN(t, b, 20, 0)
		readN(t, b, 10, 0)
		dst := make([]*entry.Entry, 20)
		_, n, err := b.Read(dst)
		require.NoError(t, err)
		require.Equal(t, 10, n)
	})

	t.Run("Write10kRandom", func(t *testing.T) {
		t.Parallel()
		rand.Seed(time.Now().Unix())
		for i := 0; i < 10; i++ {
			seed := rand.Int63()
			t.Run(strconv.Itoa(int(seed)), func(t *testing.T) {
				t.Parallel()
				r := rand.New(rand.NewSource(seed))

				b := NewMemoryBuffer(1 << 16)

				writes := 0
				reads := 0

				for i := 0; i < 10000; i++ {
					j := r.Int() % 1000
					switch {
					case j < 900:
						writeN(t, b, 1, writes)
						writes++
					default:
						readCount := (writes - reads) / 2
						f := readN(t, b, readCount, reads)
						if j%2 == 0 {
							f()
						}
						reads += readCount
					}
				}
			})
		}
	})

}

func BenchmarkMemoryBuffer(b *testing.B) {
	b.Run("AddReadWait100", func(b *testing.B) {
		buffer := NewMemoryBuffer(1 << 10)
		var wg sync.WaitGroup

		wg.Add(1)
		go func() {
			defer wg.Done()
			e := entry.New()
			e.Record = "test log"
			ctx := context.Background()
			for i := 0; i < b.N; i++ {
				panicOnErr(buffer.Add(ctx, e))
			}
		}()

		wg.Add(1)
		go func() {
			defer wg.Done()
			dst := make([]*entry.Entry, 100)
			for i := 0; i < b.N; {
				flush, n, err := buffer.ReadWait(dst, time.After(50*time.Millisecond))
				panicOnErr(err)
				i += n
				go func() {
					time.Sleep(50 * time.Millisecond)
					flush()
				}()
			}
		}()

		wg.Wait()
	})
}

func panicOnErr(err error) {
	if err != nil {
		panic(err)
	}
}
