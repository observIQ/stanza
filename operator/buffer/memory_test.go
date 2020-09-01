package buffer

import (
	"context"
	"math/rand"
	"strconv"
	"sync"
	"time"

	"testing"

	"github.com/observiq/stanza/entry"
	"github.com/observiq/stanza/testutil"
	"github.com/stretchr/testify/require"
)

func newMemoryBuffer(t testing.TB) *MemoryBuffer {
	b, err := NewMemoryBufferConfig().Build(testutil.NewBuildContext(t), "test")
	require.NoError(t, err)
	return b.(*MemoryBuffer)
}

func TestMemoryBuffer(t *testing.T) {
	t.Run("Simple", func(t *testing.T) {
		t.Parallel()
		b := newMemoryBuffer(t)
		writeN(t, b, 1, 0)
		readN(t, b, 1, 0)
	})

	t.Run("Write2Read1Read1", func(t *testing.T) {
		t.Parallel()
		b := newMemoryBuffer(t)
		writeN(t, b, 2, 0)
		readN(t, b, 1, 0)
		readN(t, b, 1, 1)
	})

	t.Run("Write20Read10Read10", func(t *testing.T) {
		t.Parallel()
		b := newMemoryBuffer(t)
		writeN(t, b, 20, 0)
		readN(t, b, 10, 0)
		readN(t, b, 10, 10)
	})

	t.Run("SingleReadWaitMultipleWrites", func(t *testing.T) {
		t.Parallel()
		b := newMemoryBuffer(t)
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
		b := newMemoryBuffer(t)
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
		b := newMemoryBuffer(t)
		writeN(t, b, 10, 0)
		readN(t, b, 10, 0)
		dst := make([]*entry.Entry, 10)
		_, n, err := b.Read(dst)
		require.NoError(t, err)
		require.Equal(t, 0, n)
	})

	t.Run("Write20Read10Read10Unfull", func(t *testing.T) {
		t.Parallel()
		b := newMemoryBuffer(t)
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

				b := newMemoryBuffer(t)

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

	t.Run("CloseReadUnflushed", func(t *testing.T) {
		t.Parallel()
		buildContext := testutil.NewBuildContext(t)
		b, err := NewMemoryBufferConfig().Build(buildContext, "test")
		require.NoError(t, err)

		writeN(t, b, 20, 0)
		readN(t, b, 5, 0)
		flushN(t, b, 5, 5)
		readN(t, b, 5, 10)

		err = b.Close()
		require.NoError(t, err)

		b2, err := NewMemoryBufferConfig().Build(buildContext, "test")
		require.NoError(t, err)

		readN(t, b2, 5, 0)
		readN(t, b2, 10, 10)
	})

}

func BenchmarkMemoryBuffer(b *testing.B) {
	buffer := newMemoryBuffer(b)
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
		dst := make([]*entry.Entry, 1000)
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
}
