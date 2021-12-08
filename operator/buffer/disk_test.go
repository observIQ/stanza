package buffer

import (
	"context"
	"errors"
	"fmt"
	"io"
	"math/rand"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/open-telemetry/opentelemetry-log-collection/entry"
	"github.com/open-telemetry/opentelemetry-log-collection/operator/helper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/sync/errgroup"
)

func TestDiskBufferBuild(t *testing.T) {
	testCases := []struct {
		desc     string
		testFunc func(*testing.T)
	}{
		{
			desc: "Fails if path is empty",
			testFunc: func(t *testing.T) {
				t.Parallel()
				cfg := NewDiskBufferConfig()
				_, err := cfg.Build()

				require.True(t, errors.Is(err, os.ErrNotExist), "did not get ErrNotExist for empty file path")
			},
		},
		{
			desc: "Builds with uncreated file",
			testFunc: func(t *testing.T) {
				t.Parallel()
				cfg := NewDiskBufferConfig()
				cfg.Path = randomFilePath("uncreated-file")
				defer func() { _ = os.RemoveAll(cfg.Path) }()

				db, err := cfg.Build()
				require.NoError(t, err)

				db.Close()
			},
		},
		{
			desc: "Builds with same file path twice",
			testFunc: func(t *testing.T) {
				t.Parallel()
				cfg := NewDiskBufferConfig()
				cfg.Path = randomFilePath("builds-twice")
				defer func() { _ = os.RemoveAll(cfg.Path) }()

				db1, err := cfg.Build()
				require.NoError(t, err)

				db1.Close()

				db2, err := cfg.Build()
				require.NoError(t, err)

				db2.Close()
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.desc, tc.testFunc)
	}
}

func TestDiskBufferAdd(t *testing.T) {
	testCases := []struct {
		desc     string
		testFunc func(*testing.T)
	}{
		{
			desc: "Can add entry to buffer",
			testFunc: func(t *testing.T) {
				t.Parallel()
				cfg := NewDiskBufferConfig()
				cfg.Path = randomFilePath("add-entry")
				defer func() { _ = os.RemoveAll(cfg.Path) }()

				db, err := cfg.Build()
				require.NoError(t, err)

				entry := entry.New()
				err = db.Add(context.Background(), entry)

				require.NoError(t, err)

				db.Close()
			},
		},
		{
			desc: "Returns err if entry cannot fit within maxDiskSize",
			testFunc: func(t *testing.T) {
				t.Parallel()
				cfg := NewDiskBufferConfig()
				cfg.Path = randomFilePath("zero-max-disk-size")
				defer func() { _ = os.RemoveAll(cfg.Path) }()

				cfg.MaxSize = 0

				db, err := cfg.Build()
				require.NoError(t, err)

				entry := entry.New()
				err = db.Add(context.Background(), entry)
				require.Error(t, err)

				db.Close()
			},
		},
		{
			desc: "Blocks if buffer is full, can be cancelled by context",
			testFunc: func(t *testing.T) {
				t.Parallel()
				cfg := NewDiskBufferConfig()
				cfg.Path = randomFilePath("block-if-full")
				defer func() { _ = os.RemoveAll(cfg.Path) }()

				entry := entry.New()

				buf := make([]byte, 0)
				buf, err := marshalEntry(buf, entry)
				require.NoError(t, err)

				cfg.MaxSize = helper.ByteSize(len(buf))

				db, err := cfg.Build()
				require.NoError(t, err)
				defer db.Close()

				err = db.Add(context.Background(), entry)
				require.NoError(t, err)

				done := make(chan struct{})
				ctx, cancel := context.WithCancel(context.Background())

				go func() {
					err = db.Add(ctx, entry)
					assert.True(t, errors.Is(err, context.Canceled), "Did not get context cancelled as error from add")
					done <- struct{}{}
				}()

				<-time.After(250 * time.Millisecond)
				cancel()

				select {
				case <-done:
				case <-time.After(250 * time.Millisecond):
					t.Error("Timed out while waiting for ctx cancel to return")
				}
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.desc, tc.testFunc)
	}
}

func TestDiskBufferRead(t *testing.T) {
	testCases := []struct {
		desc     string
		testFunc func(*testing.T)
	}{
		{
			desc: "Can read entry from buffer",
			testFunc: func(t *testing.T) {
				t.Parallel()
				cfg := NewDiskBufferConfig()
				cfg.Path = randomFilePath("read-entry")
				defer func() { _ = os.RemoveAll(cfg.Path) }()

				db, err := cfg.Build()
				require.NoError(t, err)
				defer db.Close()

				e := entry.New()
				// Set timestamp; Serializing this doesn't capture monotonic time component,
				// so it'll fail to equal the output timestamp exactly if we use time.Now().
				// This is expected.
				e.Timestamp = time.Date(2012, 1, 23, 14, 2, 1, 21, time.UTC)

				err = db.Add(context.Background(), e)
				require.NoError(t, err)

				rEntries, err := db.Read(context.Background())
				require.NoError(t, err)
				require.Len(t, rEntries, 1)
				require.Equal(t, *e, *rEntries[0])
			},
		},
		{
			desc: "Can read multiple entries from buffer",
			testFunc: func(t *testing.T) {
				t.Parallel()
				cfg := NewDiskBufferConfig()
				cfg.Path = randomFilePath("read-multiple-entries")
				defer func() { _ = os.RemoveAll(cfg.Path) }()

				db, err := cfg.Build()
				require.NoError(t, err)
				defer db.Close()

				e1 := entry.New()
				e2 := entry.New()
				e3 := entry.New()
				// Set timestamp; Serializing this doesn't capture monotonic time component,
				// so it'll fail to equal the output timestamp exactly if we use time.Now().
				// This is expected.
				e1.Timestamp = time.Date(2012, 1, 23, 14, 2, 1, 21, time.UTC)
				e2.Timestamp = time.Date(2012, 2, 23, 14, 2, 1, 21, time.UTC)
				e3.Timestamp = time.Date(2012, 3, 23, 14, 2, 1, 21, time.UTC)

				err = db.Add(context.Background(), e1)
				require.NoError(t, err)

				err = db.Add(context.Background(), e2)
				require.NoError(t, err)

				err = db.Add(context.Background(), e3)
				require.NoError(t, err)

				rEntries, err := db.Read(context.Background())
				require.NoError(t, err)
				require.Len(t, rEntries, 3)
				require.Equal(t, *e1, *rEntries[0])
				require.Equal(t, *e2, *rEntries[1])
				require.Equal(t, *e3, *rEntries[2])
			},
		},
		{
			desc: "Write happens after read",
			testFunc: func(t *testing.T) {
				t.Parallel()
				cfg := NewDiskBufferConfig()
				cfg.Path = randomFilePath("write-after-read")
				defer func() { _ = os.RemoveAll(cfg.Path) }()

				db, err := cfg.Build()
				require.NoError(t, err)
				defer db.Close()

				e := entry.New()
				// Set timestamp; Serializing this doesn't capture monotonic time component,
				// so it'll fail to equal the output timestamp exactly if we use time.Now().
				// This is expected.
				e.Timestamp = time.Date(2012, 1, 23, 14, 2, 1, 21, time.UTC)

				var entryChan = make(chan []*entry.Entry)
				go func() {
					rEntries, err := db.Read(context.Background())
					assert.NoError(t, err)
					entryChan <- rEntries
				}()

				<-time.After(100 * time.Millisecond)

				err = db.Add(context.Background(), e)
				require.NoError(t, err)

				rEntries := <-entryChan
				require.NoError(t, err)
				require.Len(t, rEntries, 1)
				require.Equal(t, *e, *rEntries[0])
			},
		},
		{
			desc: "Context gets cancelled",
			testFunc: func(t *testing.T) {
				t.Parallel()
				cfg := NewDiskBufferConfig()
				cfg.Path = randomFilePath("read-context-cancelled")
				defer func() { _ = os.RemoveAll(cfg.Path) }()

				db, err := cfg.Build()
				require.NoError(t, err)
				defer db.Close()

				e := entry.New()
				// Set timestamp; Serializing this doesn't capture monotonic time component,
				// so it'll fail to equal the output timestamp exactly if we use time.Now().
				// This is expected.
				e.Timestamp = time.Date(2012, 1, 23, 14, 2, 1, 21, time.UTC)

				done := make(chan struct{})
				ctx, cancel := context.WithCancel(context.Background())

				go func() {
					_, err = db.Read(ctx)
					assert.True(t, errors.Is(err, context.Canceled), "Did not get context cancelled as error from Read")
					done <- struct{}{}
				}()

				<-time.After(250 * time.Millisecond)
				cancel()

				select {
				case <-done:
				case <-time.After(250 * time.Millisecond):
					t.Error("Timed out while waiting for ctx cancel to return")
				}
			},
		},
		{
			desc: "Entries persist to disk",
			testFunc: func(t *testing.T) {
				t.Parallel()
				cfg := NewDiskBufferConfig()
				cfg.Path = randomFilePath("read-multiple-entries-persistence")
				cfg.MaxChunkSize = 1
				defer func() { _ = os.RemoveAll(cfg.Path) }()

				db, err := cfg.Build()
				require.NoError(t, err)

				e1 := entry.New()
				e2 := entry.New()
				// Set timestamp; Serializing this doesn't capture monotonic time component,
				// so it'll fail to equal the output timestamp exactly if we use time.Now().
				// This is expected.
				e1.Timestamp = time.Date(2012, 1, 23, 14, 2, 1, 21, time.UTC)
				e2.Timestamp = time.Date(2012, 2, 23, 14, 2, 1, 21, time.UTC)

				err = db.Add(context.Background(), e1)
				require.NoError(t, err)

				err = db.Add(context.Background(), e2)
				require.NoError(t, err)

				rEntries, err := db.Read(context.Background())
				require.NoError(t, err)
				require.Len(t, rEntries, 1)
				require.Equal(t, *e1, *rEntries[0])

				_, err = db.Close()
				require.NoError(t, err)

				db, err = cfg.Build()
				require.NoError(t, err)
				defer db.Close()

				rEntries, err = db.Read(context.Background())
				require.NoError(t, err)
				require.Len(t, rEntries, 1)
				require.Equal(t, *e2, *rEntries[0])
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.desc, tc.testFunc)
	}
}

func TestDiskBufferCompact(t *testing.T) {
	testCases := []struct {
		desc     string
		testFunc func(*testing.T)
	}{
		{
			desc: "Test compact with entry present",
			testFunc: func(t *testing.T) {
				t.Parallel()
				cfg := NewDiskBufferConfig()
				cfg.Path = randomFilePath("compact-with-entry")
				cfg.MaxChunkSize = 1
				defer func() { _ = os.RemoveAll(cfg.Path) }()

				db, err := cfg.Build()
				require.NoError(t, err)
				defer db.Close()

				e := entry.New()

				// Set timestamp; Serializing this doesn't capture monotonic time component,
				// so it'll fail to equal the output timestamp exactly if we use time.Now().
				// This is expected.
				e.Timestamp = time.Date(2012, 1, 23, 14, 2, 1, 21, time.UTC)

				err = db.Add(context.Background(), e)
				require.NoError(t, err)
				err = db.Add(context.Background(), e)
				require.NoError(t, err)

				rEntries, err := db.Read(context.Background())
				require.NoError(t, err)
				require.Len(t, rEntries, 1)
				require.Equal(t, *e, *rEntries[0])

				// Manually trigger a compact
				err = db.(*DiskBuffer).compact()
				require.NoError(t, err)

				buf := make([]byte, 0)
				buf, err = marshalEntry(buf, e)
				require.NoError(t, err)

				fLen, err := db.(*DiskBuffer).f.Seek(0, io.SeekEnd)
				require.NoError(t, err)
				require.Equal(t, DiskBufferMetadataBinarySize+len(buf), int(fLen))

				rEntries, err = db.Read(context.Background())
				require.NoError(t, err)
				require.Len(t, rEntries, 1)
				require.Equal(t, *e, *rEntries[0])
			},
		},
		{
			desc: "Test compact while empty",
			testFunc: func(t *testing.T) {
				t.Parallel()
				cfg := NewDiskBufferConfig()
				cfg.Path = randomFilePath("compact-empty")
				cfg.MaxChunkSize = 1
				defer func() { _ = os.RemoveAll(cfg.Path) }()

				db, err := cfg.Build()
				require.NoError(t, err)
				defer db.Close()

				e := entry.New()

				// Set timestamp; Serializing this doesn't capture monotonic time component,
				// so it'll fail to equal the output timestamp exactly if we use time.Now().
				// This is expected.
				e.Timestamp = time.Date(2012, 1, 23, 14, 2, 1, 21, time.UTC)

				err = db.Add(context.Background(), e)
				require.NoError(t, err)

				rEntries, err := db.Read(context.Background())
				require.NoError(t, err)
				require.Len(t, rEntries, 1)
				require.Equal(t, *e, *rEntries[0])

				// Manually trigger a compact
				err = db.(*DiskBuffer).compact()
				require.NoError(t, err)

				fLen, err := db.(*DiskBuffer).f.Seek(0, io.SeekEnd)
				require.NoError(t, err)
				require.Equal(t, int(fLen), DiskBufferMetadataBinarySize)
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.desc, tc.testFunc)
	}
}

func TestDiskBufferClose(t *testing.T) {
	testCases := []struct {
		desc     string
		testFunc func(*testing.T)
	}{
		{
			desc: "Cannot Add or Read after Close",
			testFunc: func(t *testing.T) {
				t.Parallel()
				cfg := NewDiskBufferConfig()
				cfg.Path = randomFilePath("operate-after-close")
				defer func() { _ = os.RemoveAll(cfg.Path) }()

				db, err := cfg.Build()
				require.NoError(t, err)

				_, err = db.Close()
				require.NoError(t, err)

				e := entry.New()
				err = db.Add(context.Background(), e)
				require.ErrorIs(t, err, ErrBufferClosed)

				_, err = db.Read(context.Background())
				require.ErrorIs(t, err, ErrBufferClosed)
			},
		},
		{
			desc: "Multiple Closes Return No Error",
			testFunc: func(t *testing.T) {
				t.Parallel()
				cfg := NewDiskBufferConfig()
				cfg.Path = randomFilePath("close-after-close")
				defer func() { _ = os.RemoveAll(cfg.Path) }()

				db, err := cfg.Build()
				require.NoError(t, err)

				_, err = db.Close()
				require.NoError(t, err)

				_, err = db.Close()
				require.NoError(t, err)
			},
		},
		{
			desc: "Currently running Adds will error",
			testFunc: func(t *testing.T) {
				t.Parallel()
				cfg := NewDiskBufferConfig()
				cfg.Path = randomFilePath("close-stops-add")
				defer func() { _ = os.RemoveAll(cfg.Path) }()

				entry := entry.New()

				buf := make([]byte, 0)
				buf, err := marshalEntry(buf, entry)
				require.NoError(t, err)

				cfg.MaxSize = helper.ByteSize(len(buf))

				db, err := cfg.Build()
				require.NoError(t, err)

				err = db.Add(context.Background(), entry)
				require.NoError(t, err)

				done := make(chan struct{})

				go func() {
					err = db.Add(context.Background(), entry)
					assert.ErrorIs(t, err, ErrBufferClosed)
					done <- struct{}{}
				}()

				<-time.After(100 * time.Millisecond)
				db.Close()

				select {
				case <-done:
				case <-time.After(250 * time.Millisecond):
					t.Error("Timed out while waiting for Add to return")
				}
			},
		},
		{
			desc: "Currently running Reads will error",
			testFunc: func(t *testing.T) {
				t.Parallel()
				cfg := NewDiskBufferConfig()
				cfg.Path = randomFilePath("close-stops-read")
				defer func() { _ = os.RemoveAll(cfg.Path) }()

				db, err := cfg.Build()
				require.NoError(t, err)

				done := make(chan struct{})

				go func() {
					_, err = db.Read(context.Background())
					assert.ErrorIs(t, err, ErrBufferClosed)
					done <- struct{}{}
				}()

				<-time.After(100 * time.Millisecond)
				db.Close()

				select {
				case <-done:
				case <-time.After(250 * time.Millisecond):
					t.Error("Timed out while waiting for Add to return")
				}
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.desc, tc.testFunc)
	}
}

func TestDiskBufferConcurrency(t *testing.T) {
	var (
		numEntries = 10000
		timeout    = 15 * time.Second
	)

	testCases := []struct {
		writers int
		readers int
	}{
		{
			readers: 1,
			writers: 1,
		},
		{
			readers: 3,
			writers: 1,
		},
		{
			readers: 1,
			writers: 3,
		},
		{
			readers: 3,
			writers: 3,
		},
		{
			readers: 12,
			writers: 1,
		},
		{
			readers: 1,
			writers: 12,
		},
		{
			readers: 12,
			writers: 12,
		},
	}

	for _, testCase := range testCases {
		t.Run(fmt.Sprintf("%d-Readers-%d-Writers", testCase.readers, testCase.writers), func(t *testing.T) {
			t.Parallel()
			cfg := NewDiskBufferConfig()
			cfg.Path = randomFilePath("concurrency-test")
			cfg.MaxSize = 1 << 20 // 1 meg
			cfg.MaxChunkDelay.Duration = 25 * time.Millisecond
			t.Logf(`Using path %s`, cfg.Path)
			defer func() { _ = os.RemoveAll(cfg.Path) }()

			buf, err := cfg.Build()
			require.NoError(t, err)

			timeoutCtx, cancel := context.WithTimeout(context.Background(), timeout)
			defer cancel()
			errGrp, ctx := errgroup.WithContext(timeoutCtx)
			var readCnt int64 = 0
			// Spin off readers
			for i := 0; i < testCase.readers; i++ {
				errGrp.Go(
					func() error {
						for {
							entries, err := buf.Read(ctx)
							if err != nil {
								return err
							}

							updatedCnt := atomic.AddInt64(&readCnt, int64(len(entries)))
							if updatedCnt == int64(numEntries) {
								return nil
							}
						}
					},
				)
			}

			// Spin off writers
			entriesPerWriter := numEntries / testCase.writers
			for i := 0; i < testCase.writers; i++ {
				if i == testCase.writers-1 {
					entriesPerWriter = entriesPerWriter + (numEntries % testCase.writers)
				}

				entries := randomEntries(entriesPerWriter)
				errGrp.Go(
					func() error {
						for _, e := range entries {
							err := buf.Add(ctx, e)
							if err != nil {
								return err
							}
						}
						return nil
					},
				)
			}

			err = errGrp.Wait()
			require.NoError(t, err)
		})
	}
}

func BenchmarkDiskBuffer(b *testing.B) {
	var (
		numEntries    = 1000
		timeout       = 15 * time.Second
		maxSize       = 1 << 14 // 16 Kb file
		maxChunkDelay = 25 * time.Millisecond
	)

	testCases := []struct {
		writers int
		readers int
	}{
		{
			readers: 1,
			writers: 1,
		},
		{
			readers: 3,
			writers: 1,
		},
		{
			readers: 1,
			writers: 3,
		},
		{
			readers: 3,
			writers: 3,
		},
		{
			readers: 12,
			writers: 1,
		},
		{
			readers: 1,
			writers: 12,
		},
		{
			readers: 12,
			writers: 12,
		},
	}

	for _, testCase := range testCases {
		b.Run(fmt.Sprintf("Benchmark1KEntries-%d-Readers-%d-Writers", testCase.readers, testCase.writers), func(b *testing.B) {
			cfg := NewDiskBufferConfig()
			cfg.Path = randomFilePath("concurrency-test")
			cfg.MaxSize = helper.ByteSize(maxSize)
			cfg.MaxChunkDelay.Duration = maxChunkDelay
			defer func() { _ = os.RemoveAll(cfg.Path) }()

			buf, err := cfg.Build()
			require.NoError(b, err)

			entrySets := make([][]*entry.Entry, testCase.writers)
			entriesPerWriter := numEntries / testCase.writers
			for i := 0; i < testCase.writers; i++ {
				if i == testCase.writers-1 {
					entriesPerWriter = entriesPerWriter + (numEntries % testCase.writers)
				}

				entrySets[i] = randomEntries(entriesPerWriter)
			}

			b.ResetTimer()
			b.StopTimer()
			for i := 0; i < b.N; i++ {
				timeoutCtx, cancel := context.WithTimeout(context.Background(), timeout)
				errGrp, ctx := errgroup.WithContext(timeoutCtx)
				var readCnt int64 = 0

				// Spin off readers
				for i := 0; i < testCase.readers; i++ {
					errGrp.Go(
						func() error {
							for {
								entries, err := buf.Read(ctx)
								if err != nil {
									return err
								}

								updatedCnt := atomic.AddInt64(&readCnt, int64(len(entries)))
								if updatedCnt == int64(numEntries) {
									return nil
								}
							}
						},
					)
				}

				b.StartTimer()
				// Spin off writers

				for i := 0; i < testCase.writers; i++ {
					writer := i
					errGrp.Go(
						func() error {
							for _, e := range entrySets[writer] {
								err := buf.Add(ctx, e)
								if err != nil {
									return err
								}
							}
							return nil
						},
					)
				}

				err = errGrp.Wait()
				b.StopTimer()
				require.NoError(b, err)
				cancel()
			}
		})
	}
}

func randomFilePath(prefix string) string {
	return filepath.Join(os.TempDir(), prefix+randomString(16))
}

const alphabet = "abcdefghijklmnopqrstuvwxyz"

func randomString(l int) string {
	b := strings.Builder{}
	b.Grow(int(l))

	for i := 0; i < l; i++ {
		c := rand.Int() % len(alphabet)
		b.Write([]byte{alphabet[c]})
	}

	return b.String()
}

func randomEntries(n int) []*entry.Entry {
	entries := make([]*entry.Entry, 0, n)
	for i := 0; i < n; i++ {
		entries = append(entries, randomEntry())
	}
	return entries
}

func randomEntry() *entry.Entry {
	e := entry.New()
	e.Timestamp = time.Unix(rand.Int63n(1638884759), rand.Int63n(1e9)).UTC()
	e.Body = map[string]interface{}{
		"msg": randomString(16),
	}
	e.Attributes = map[string]string{
		"file": randomString(19),
	}

	return e
}
