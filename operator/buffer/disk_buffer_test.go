package buffer

import (
	"context"
	"strconv"
	"time"

	"testing"

	"github.com/observiq/carbon/entry"
	"github.com/observiq/carbon/internal/testutil"
	"github.com/stretchr/testify/require"
)

func intEntry(i int) *entry.Entry {
	e := entry.New()
	e.Timestamp = time.Date(2006, 01, 02, 03, 04, 05, 06, time.UTC)
	e.Record = float64(i)
	return e
}

func writeN(t testing.TB, buffer *DiskBuffer, n int) {
	ctx := context.Background()
	for i := 0; i < n; i++ {
		err := buffer.Add(ctx, intEntry(i))
		require.NoError(t, err)
	}
}

func batchN(t testing.TB, buffer *DiskBuffer, n int) {
	entries := make([]*entry.Entry, n)
	for i := 0; i < n; i++ {
		entries[i] = intEntry(i)
	}
	ctx := context.Background()
	err := buffer.BatchAdd(ctx, entries)
	require.NoError(t, err)
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
		b := openBuffer(t)
		writeN(t, b, 1)
		readN(t, b, 1, 0)
	})

	t.Run("Write20Read10Read10", func(t *testing.T) {
		b := openBuffer(t)
		writeN(t, b, 20)
		readN(t, b, 10, 0)
		readN(t, b, 10, 10)
	})

	t.Run("Write10Read10Read0", func(t *testing.T) {
		b := openBuffer(t)
		writeN(t, b, 10)
		readN(t, b, 10, 0)
		dst := make([]*entry.Entry, 10)
		_, n, err := b.Read(dst)
		require.NoError(t, err)
		require.Equal(t, 0, n)
	})

	t.Run("Write20Read10Read10Unfull", func(t *testing.T) {
		b := openBuffer(t)
		writeN(t, b, 20)
		readN(t, b, 10, 0)
		dst := make([]*entry.Entry, 20)
		_, n, err := b.Read(dst)
		require.NoError(t, err)
		require.Equal(t, 10, n)
	})

	t.Run("Write20Read10CompactRead10", func(t *testing.T) {
		b := openBuffer(t)
		writeN(t, b, 20)
		flushN(t, b, 10, 0)
		compact(t, b)
		readN(t, b, 10, 10)
	})

	t.Run("Batch20Read10Read10", func(t *testing.T) {
		b := openBuffer(t)
		batchN(t, b, 20)
		readN(t, b, 10, 0)
		readN(t, b, 10, 10)
	})

	t.Run("Batch10Read10Read0", func(t *testing.T) {
		b := openBuffer(t)
		batchN(t, b, 10)
		readN(t, b, 10, 0)
		dst := make([]*entry.Entry, 10)
		_, n, err := b.Read(dst)
		require.NoError(t, err)
		require.Equal(t, 0, n)
	})

	t.Run("Batch20Read10Read10Unfull", func(t *testing.T) {
		b := openBuffer(t)
		batchN(t, b, 20)
		readN(t, b, 10, 0)
		dst := make([]*entry.Entry, 20)
		_, n, err := b.Read(dst)
		require.NoError(t, err)
		require.Equal(t, 10, n)
	})

	t.Run("Batch20Read10CompactRead10", func(t *testing.T) {
		b := openBuffer(t)
		batchN(t, b, 20)
		flushN(t, b, 10, 0)
		compact(t, b)
		readN(t, b, 10, 10)
	})

	t.Run("Write20Read10CloseRead20", func(t *testing.T) {
		b := NewDiskBuffer()
		dir := testutil.NewTempDir(t)
		err := b.Open(dir)
		require.NoError(t, err)

		batchN(t, b, 20)
		readN(t, b, 10, 0)
		b.Close()

		b2 := NewDiskBuffer()
		err = b2.Open(dir)
		require.NoError(t, err)
		readN(t, b2, 20, 0)
	})

	t.Run("Write20Flush10CloseRead20", func(t *testing.T) {
		b := NewDiskBuffer()
		dir := testutil.NewTempDir(t)
		err := b.Open(dir)
		require.NoError(t, err)

		batchN(t, b, 20)
		flushN(t, b, 10, 0)
		b.Close()

		b2 := NewDiskBuffer()
		err = b2.Open(dir)
		require.NoError(t, err)
		readN(t, b2, 10, 10)
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

func BenchmarkDiskBufferBatchWrite(b *testing.B) {
	for _, batchSize := range []int{1, 5, 10, 50, 100} {
		b.Run(strconv.Itoa(batchSize), func(b *testing.B) {
			buffer := openBuffer(b)

			e := entry.New()
			ctx := context.Background()
			batch := make([]*entry.Entry, batchSize)
			for i := 0; i < batchSize; i++ {
				batch[i] = e
			}

			b.ResetTimer()
			for i := 0; i < b.N; i += batchSize {
				buffer.BatchAdd(ctx, batch)
			}
		})
	}
}

func BenchmarkDiskBufferCompact(b *testing.B) {
	b.Run("AllFlushed", func(b *testing.B) {
		for _, n := range []int{1, 10, 100, 1000, 10000} {
			b.Run(strconv.Itoa(n), func(b *testing.B) {
				for i := 0; i < b.N; i++ {
					b.StopTimer()
					buffer := openBuffer(b)
					batchN(b, buffer, n)
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
					batchN(b, buffer, n)
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
					batchN(b, buffer, n)
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
					batchN(b, buffer, n)

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
