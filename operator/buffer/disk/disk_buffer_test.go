package disk

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

func intEntry(i int) *entry.Entry {
	e := entry.New()
	e.Timestamp = time.Date(2006, 01, 02, 03, 04, 05, 06, time.UTC)
	e.Record = float64(i)
	return e
}

func writeN(t testing.TB, buffer *DiskBuffer, n, start int) {
	ctx := context.Background()
	for i := start; i < n+start; i++ {
		err := buffer.Add(ctx, intEntry(i))
		require.NoError(t, err)
	}
}

func readN(t testing.TB, buffer *DiskBuffer, n, start int) func() {
	entries := make([]*entry.Entry, n)
	f, readCount, err := buffer.Read(entries)
	require.NoError(t, err)
	require.Equal(t, n, readCount)
	for i := 0; i < n; i++ {
		require.Equal(t, intEntry(start+i), entries[i])
	}
	return f
}

func readWaitN(t testing.TB, buffer *DiskBuffer, n, start int) func() {
	entries := make([]*entry.Entry, n)
	f, readCount, err := buffer.ReadWait(entries, time.After(time.Minute))
	require.NoError(t, err)
	require.Equal(t, n, readCount)
	for i := 0; i < n; i++ {
		require.Equal(t, intEntry(start+i), entries[i])
	}
	return f
}

func uncheckedReadN(t testing.TB, buffer *DiskBuffer, n int) func() {
	entries := make([]*entry.Entry, n)
	f, readCount, _ := buffer.Read(entries)
	require.Equal(t, n, readCount)
	return f
}

func flushN(t testing.TB, buffer *DiskBuffer, n, start int) {
	f := readN(t, buffer, n, start)
	f()
}

func uncheckedFlushN(t testing.TB, buffer *DiskBuffer, n int) {
	f := uncheckedReadN(t, buffer, n)
	f()
}

func openBuffer(t testing.TB) *DiskBuffer {
	buffer := NewDiskBuffer()
	dir := testutil.NewTempDir(t)
	err := buffer.Open(dir)
	require.NoError(t, err)
	t.Cleanup(func() { buffer.Close() })
	return buffer
}

func compact(t testing.TB, b *DiskBuffer) {
	err := b.Compact()
	require.NoError(t, err)
}

func TestDiskBuffer(t *testing.T) {
	t.Run("Simple", func(t *testing.T) {
		t.Parallel()
		b := openBuffer(t)
		writeN(t, b, 1, 0)
		readN(t, b, 1, 0)
	})

	t.Run("Write2Read1Read1", func(t *testing.T) {
		t.Parallel()
		b := openBuffer(t)
		writeN(t, b, 2, 0)
		readN(t, b, 1, 0)
		readN(t, b, 1, 1)
	})

	t.Run("Write20Read10Read10", func(t *testing.T) {
		t.Parallel()
		b := openBuffer(t)
		writeN(t, b, 20, 0)
		readN(t, b, 10, 0)
		readN(t, b, 10, 10)
	})

	t.Run("SingleReadWaitMultipleWrites", func(t *testing.T) {
		t.Parallel()
		b := openBuffer(t)
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
		b := openBuffer(t)
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
		b := openBuffer(t)
		writeN(t, b, 10, 0)
		readN(t, b, 10, 0)
		dst := make([]*entry.Entry, 10)
		_, n, err := b.Read(dst)
		require.NoError(t, err)
		require.Equal(t, 0, n)
	})

	t.Run("Write20Read10Read10Unfull", func(t *testing.T) {
		t.Parallel()
		b := openBuffer(t)
		writeN(t, b, 20, 0)
		readN(t, b, 10, 0)
		dst := make([]*entry.Entry, 20)
		_, n, err := b.Read(dst)
		require.NoError(t, err)
		require.Equal(t, 10, n)
	})

	t.Run("Write20Read10CompactRead10", func(t *testing.T) {
		t.Parallel()
		b := openBuffer(t)
		writeN(t, b, 20, 0)
		flushN(t, b, 10, 0)
		compact(t, b)
		readN(t, b, 10, 10)
	})

	t.Run("Write20Read10CloseRead20", func(t *testing.T) {
		t.Parallel()
		b := NewDiskBuffer()
		dir := testutil.NewTempDir(t)
		err := b.Open(dir)
		require.NoError(t, err)

		writeN(t, b, 20, 0)
		readN(t, b, 10, 0)
		err = b.Close()
		require.NoError(t, err)

		b2 := NewDiskBuffer()
		err = b2.Open(dir)
		require.NoError(t, err)
		readN(t, b2, 20, 0)
	})

	t.Run("Write20Flush10CloseRead20", func(t *testing.T) {
		t.Parallel()
		b := NewDiskBuffer()
		dir := testutil.NewTempDir(t)
		err := b.Open(dir)
		require.NoError(t, err)

		writeN(t, b, 20, 0)
		flushN(t, b, 10, 0)
		err = b.Close()
		require.NoError(t, err)

		b2 := NewDiskBuffer()
		err = b2.Open(dir)
		require.NoError(t, err)
		readN(t, b2, 10, 10)
	})

	t.Run("Write10kRandomFlushReadCompact", func(t *testing.T) {
		t.Parallel()
		rand.Seed(time.Now().Unix())
		for i := 0; i < 10; i++ {
			seed := rand.Int63()
			t.Run(strconv.Itoa(int(seed)), func(t *testing.T) {
				t.Parallel()
				r := rand.New(rand.NewSource(seed))

				b := NewDiskBuffer()
				dir := testutil.NewTempDir(t)
				err := b.Open(dir)
				require.NoError(t, err)

				writes := 0
				reads := 0

				for i := 0; i < 10000; i++ {
					j := r.Int() % 1000
					switch {
					case j < 900:
						writeN(t, b, 1, writes)
						writes++
					case j < 990:
						readCount := (writes - reads) / 2
						f := readN(t, b, readCount, reads)
						if j%2 == 0 {
							f()
						}
						reads += readCount
					default:
						err := b.Compact()
						require.NoError(t, err)
					}
				}
			})
		}
	})

}

func BenchmarkDiskBufferWrite(b *testing.B) {
	buffer := openBuffer(b)

	e := entry.New()
	ctx := context.Background()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		buffer.Add(ctx, e)
	}
}

func BenchmarkDiskBuffer(b *testing.B) {
	b.Run("AddReadWait100", func(b *testing.B) {
		buffer := openBuffer(b)
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

		cancel := make(chan struct{})
		done := make(chan struct{})
		go func() {
			defer close(done)
			for {
				select {
				case <-cancel:
					return
				case <-time.After(50 * time.Millisecond):
					buffer.Compact()
				}
			}
		}()

		wg.Wait()
		close(cancel)
		<-done
	})
}

func BenchmarkDiskBufferCompact(b *testing.B) {
	b.Run("AllFlushed", func(b *testing.B) {
		for _, n := range []int{1, 10, 100, 1000, 10000} {
			b.Run(strconv.Itoa(n), func(b *testing.B) {
				for i := 0; i < b.N; i++ {
					b.StopTimer()
					buffer := openBuffer(b)
					writeN(b, buffer, n, 0)
					uncheckedFlushN(b, buffer, n)
					b.StartTimer()

					err := buffer.Compact()
					require.NoError(b, err)
					buffer.Close()
				}
			})
		}
	})

	b.Run("AllRead", func(b *testing.B) {
		for _, n := range []int{1, 10, 100, 1000, 10000} {
			b.Run(strconv.Itoa(n), func(b *testing.B) {
				for i := 0; i < b.N; i++ {
					b.StopTimer()
					buffer := openBuffer(b)
					writeN(b, buffer, n, 0)
					uncheckedReadN(b, buffer, n)
					b.StartTimer()

					err := buffer.Compact()
					require.NoError(b, err)
					buffer.Close()
				}
			})
		}
	})

	b.Run("NoneRead", func(b *testing.B) {
		for _, n := range []int{1, 10, 100, 1000, 10000} {
			b.Run(strconv.Itoa(n), func(b *testing.B) {
				for i := 0; i < b.N; i++ {
					b.StopTimer()
					buffer := openBuffer(b)
					writeN(b, buffer, n, 0)
					b.StartTimer()

					err := buffer.Compact()
					require.NoError(b, err)
					buffer.Close()
				}
			})
		}
	})

	b.Run("Fragmented", func(b *testing.B) {
		for _, n := range []int{100, 1000, 10000} {
			b.Run(strconv.Itoa(n), func(b *testing.B) {
				for i := 0; i < b.N; i++ {
					b.StopTimer()
					buffer := openBuffer(b)
					writeN(b, buffer, n, 0)

					// alternate reading and flushing in batches of 10
					flush := false
					for j := 0; j < n; j += 10 {
						if flush {
							uncheckedFlushN(b, buffer, 10)
							flush = false
							continue
						}
						uncheckedReadN(b, buffer, 10)
						flush = true
					}
					b.StartTimer()

					err := buffer.Compact()
					require.NoError(b, err)
					buffer.Close()
				}
			})
		}
	})

}

func panicOnErr(err error) {
	if err != nil {
		panic(err)
	}
}

func dumpBuffer(buffer *DiskBuffer, path string) {

}
