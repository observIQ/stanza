package buffer

import (
	"io"
	"math/rand"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/observiq/stanza/v2/operator/buffer/mocks"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestOpen(t *testing.T) {
	testCases := []struct {
		desc     string
		testFunc func(*testing.T)
	}{
		{
			desc: "Test open has no error",
			testFunc: func(t *testing.T) {
				t.Parallel()
				path := randomFilePath("ring-buffer-open")
				defer os.RemoveAll(path)

				cf, err := openCircularFile(path, false, 1000)
				require.NoError(t, err)
				defer cf.Close()

				require.NotNil(t, cf)
			},
		},
		{
			desc: "Test open sync has no error",
			testFunc: func(t *testing.T) {
				t.Parallel()
				path := randomFilePath("ring-buffer-open-sync")
				defer os.RemoveAll(path)

				cf, err := openCircularFile(path, true, 1000)
				require.NoError(t, err)
				defer cf.Close()

				require.NotNil(t, cf)
			},
		},
		{
			desc: "Test directory path gives error",
			testFunc: func(t *testing.T) {
				t.Parallel()
				path := os.TempDir()

				cf, err := openCircularFile(path, true, 1000)
				require.Error(t, err)
				require.Nil(t, cf)
			},
		},
		{
			desc: "Test open with new size gives error",
			testFunc: func(t *testing.T) {
				t.Parallel()
				path := randomFilePath("ring-buffer-open-sync")

				cf, err := openCircularFile(path, true, 1000)
				require.NoError(t, err)

				err = cf.Close()
				require.NoError(t, err)

				cf, err = openCircularFile(path, true, 1001)
				require.Error(t, err)
				require.Nil(t, cf)

			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.desc, tc.testFunc)
	}
}

func TestWrite(t *testing.T) {
	testCases := []struct {
		desc     string
		testFunc func(*testing.T)
	}{
		{
			desc: "Writing has no error",
			testFunc: func(t *testing.T) {
				t.Parallel()
				path := randomFilePath("ring-buffer-write")
				defer os.RemoveAll(path)

				cf, err := openCircularFile(path, false, 1000)
				require.NoError(t, err)
				defer cf.Close()

				require.NotNil(t, cf)
				b := []byte{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12}
				n, err := cf.Write(b)
				require.NoError(t, err)
				require.Equal(t, len(b), n)
				require.Equal(t, int64(len(b)), cf.len())
			},
		},
		{
			desc: "Writing twice has no error",
			testFunc: func(t *testing.T) {
				t.Parallel()
				path := randomFilePath("ring-buffer-write-twice")
				defer os.RemoveAll(path)

				cf, err := openCircularFile(path, false, 1000)
				require.NoError(t, err)
				defer cf.Close()

				require.NotNil(t, cf)
				b := []byte{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12}

				n, err := cf.Write(b)
				require.NoError(t, err)
				require.Equal(t, len(b), n)
				require.Equal(t, int64(len(b)), cf.len())

				n, err = cf.Write(b)
				require.NoError(t, err)
				require.Equal(t, len(b), n)
				require.Equal(t, int64(len(b)*2), cf.len())
			},
		},
		{
			desc: "Writing rb.Len() bytes is valid",
			testFunc: func(t *testing.T) {
				t.Parallel()
				path := randomFilePath("ring-buffer-write-full-length")
				defer os.RemoveAll(path)

				b := []byte{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12}

				cf, err := openCircularFile(path, false, int64(len(b)))
				require.NoError(t, err)
				defer cf.Close()

				require.NotNil(t, cf)

				n, err := cf.Write(b)
				require.NoError(t, err)
				require.Equal(t, len(b), n)
				require.Equal(t, int64(len(b)), cf.len())
				require.True(t, cf.Full)
			},
		},
		{
			desc: "Writing when full gives EOF",
			testFunc: func(t *testing.T) {
				t.Parallel()
				path := randomFilePath("ring-buffer-write-full-length")
				defer os.RemoveAll(path)

				b := []byte{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12}

				cf, err := openCircularFile(path, false, int64(len(b)))
				require.NoError(t, err)
				defer cf.Close()

				require.NotNil(t, cf)

				n, err := cf.Write(b)
				require.NoError(t, err)
				require.Equal(t, len(b), n)
				require.Equal(t, int64(len(b)), cf.len())
				require.True(t, cf.Full)

				n, err = cf.Write(b)
				require.ErrorIs(t, err, io.EOF)
				require.Equal(t, 0, n)
				require.Equal(t, int64(len(b)), cf.len())
				require.True(t, cf.Full)
			},
		},
		{
			desc: "Writing over boundary is valid",
			testFunc: func(t *testing.T) {
				t.Parallel()
				path := randomFilePath("ring-buffer-write-boundary")
				defer os.RemoveAll(path)

				b := []byte{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12}

				cf, err := openCircularFile(path, false, int64(len(b)))
				require.NoError(t, err)
				defer cf.Close()

				require.NotNil(t, cf)

				cf.Start = int64(len(b) / 2)
				cf.End = cf.Start
				cf.ReadPtr = cf.Start

				n, err := cf.Write(b)
				require.NoError(t, err)
				require.Equal(t, len(b), n)
				require.Equal(t, int64(len(b)), cf.len())
				require.True(t, cf.Full)
			},
		},
		{
			desc: "Trying to write over buffer returns EOF",
			testFunc: func(t *testing.T) {
				t.Parallel()
				path := randomFilePath("ring-buffer-write-overflow")
				defer os.RemoveAll(path)

				b := []byte{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12}

				cf, err := openCircularFile(path, false, int64(len(b)-1))
				require.NoError(t, err)
				defer cf.Close()

				require.NotNil(t, cf)

				n, err := cf.Write(b)
				require.ErrorIs(t, err, io.EOF)
				require.Equal(t, len(b)-1, n)
				require.Equal(t, int64(len(b)-1), cf.len())
				require.True(t, cf.Full)
			},
		},
		{
			desc: "First write fails",
			testFunc: func(t *testing.T) {
				t.Parallel()
				path := randomFilePath("ring-buffer-write-fails")
				defer os.RemoveAll(path)

				b := []byte{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12}

				f := &mocks.FileLike{}
				f.On("Seek", mock.Anything, mock.Anything).Return(int64(0), nil)
				f.On("Write", b).Return(2, io.ErrUnexpectedEOF)

				rb := circularFile{
					Size: int64(len(b)),
					f:    f,
				}

				n, err := rb.Write(b)
				require.ErrorIs(t, err, io.ErrUnexpectedEOF)
				require.Equal(t, 2, n)
				require.Equal(t, int64(2), rb.len())
				require.Equal(t, int64(2), rb.End)
				require.False(t, rb.Full)
			},
		},
		{
			desc: "Second write fails",
			testFunc: func(t *testing.T) {
				t.Parallel()
				path := randomFilePath("ring-buffer-write2-fails")
				defer os.RemoveAll(path)

				b := []byte{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12}
				firstWriteAmount := len(b) - len(b)/2
				firstWriteSlice := b[:firstWriteAmount]
				secondWriteSlice := b[firstWriteAmount:]

				f := &mocks.FileLike{}
				f.On("Seek", mock.Anything, mock.Anything).Return(int64(0), nil)
				f.On("Write", firstWriteSlice).Return(len(firstWriteSlice), nil)
				f.On("Write", secondWriteSlice).Return(1, io.ErrShortWrite)

				cf := circularFile{
					Size: int64(len(b)),
					f:    f,
				}

				cf.Start = int64(len(b) / 2)
				cf.End = cf.Start
				cf.ReadPtr = cf.Start

				n, err := cf.Write(b)
				require.ErrorIs(t, err, io.ErrShortWrite)
				require.Equal(t, len(firstWriteSlice)+1, n)
				require.Equal(t, int64(len(firstWriteSlice))+1, cf.len())
				require.Equal(t, int64(1), cf.End)
				require.False(t, cf.Full)
			},
		},
		{
			desc: "First seek fails",
			testFunc: func(t *testing.T) {
				t.Parallel()
				path := randomFilePath("ring-buffer-seek-fails")
				defer os.RemoveAll(path)

				b := []byte{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12}

				f := &mocks.FileLike{}
				f.On("Seek", mock.Anything, mock.Anything).Return(int64(0), io.ErrClosedPipe)

				cf := circularFile{
					Size: int64(len(b)),
					f:    f,
				}

				n, err := cf.Write(b)
				require.ErrorIs(t, err, io.ErrClosedPipe)
				require.Equal(t, 0, n)
				require.Equal(t, int64(0), cf.len())
				require.Equal(t, int64(0), cf.End)
				require.False(t, cf.Full)
			},
		},
		{
			desc: "Second seek fails",
			testFunc: func(t *testing.T) {
				t.Parallel()
				path := randomFilePath("ring-buffer-seek2-fails")
				defer os.RemoveAll(path)

				b := []byte{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12}
				firstWriteAmount := len(b) - len(b)/2
				firstWriteSlice := b[:firstWriteAmount]
				secondWriteSlice := b[firstWriteAmount:]

				f := &mocks.FileLike{}
				f.On("Seek", int64(len(b)/2), io.SeekStart).Return(int64(0), nil)
				f.On("Seek", int64(0), io.SeekStart).Return(int64(0), io.ErrClosedPipe)
				f.On("Write", firstWriteSlice).Return(len(firstWriteSlice), nil)
				f.On("Write", secondWriteSlice).Return(len(secondWriteSlice), nil)

				cf := circularFile{
					Size: int64(len(b)),
					f:    f,
				}

				cf.Start = int64(len(b) / 2)
				cf.End = cf.Start
				cf.ReadPtr = cf.Start

				n, err := cf.Write(b)
				require.ErrorIs(t, err, io.ErrClosedPipe)
				require.Equal(t, len(firstWriteSlice), n)
				require.Equal(t, int64(len(firstWriteSlice)), cf.len())
				require.Equal(t, int64(0), cf.End)
				require.False(t, cf.Full)
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.desc, tc.testFunc)
	}
}

func TestRead(t *testing.T) {
	testCases := []struct {
		desc     string
		testFunc func(*testing.T)
	}{
		{
			desc: "Reading has no error",
			testFunc: func(t *testing.T) {
				t.Parallel()
				path := randomFilePath("ring-buffer-read")
				defer os.RemoveAll(path)

				cf, err := openCircularFile(path, false, 1000)
				require.NoError(t, err)
				defer cf.Close()

				require.NotNil(t, cf)
				b := []byte{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12}
				n, err := cf.Write(b)
				require.NoError(t, err)
				require.Equal(t, len(b), n)
				require.Equal(t, int64(len(b)), cf.len())

				bufOut := make([]byte, len(b))
				n, err = cf.Read(bufOut)
				require.NoError(t, err)
				require.Equal(t, len(bufOut), n)
				require.Equal(t, b, bufOut)
			},
		},
		{
			desc: "Reading twice has no error",
			testFunc: func(t *testing.T) {
				t.Parallel()
				path := randomFilePath("ring-buffer-read")
				defer os.RemoveAll(path)

				cf, err := openCircularFile(path, false, 1000)
				require.NoError(t, err)
				defer cf.Close()

				require.NotNil(t, cf)
				b := []byte{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13}
				n, err := cf.Write(b)
				require.NoError(t, err)
				require.Equal(t, len(b), n)
				require.Equal(t, int64(len(b)), cf.len())

				bufOut := make([]byte, len(b))
				n, err = cf.Read(bufOut[:7])
				require.NoError(t, err)
				require.Equal(t, 7, n)
				require.Equal(t, b[:7], bufOut[:7])

				n, err = cf.Read(bufOut[7:])
				require.NoError(t, err)
				require.Equal(t, 7, n)
				require.Equal(t, b, bufOut)
			},
		},
		{
			desc: "Reading rb.Len() bytes is valid",
			testFunc: func(t *testing.T) {
				t.Parallel()
				path := randomFilePath("ring-buffer-read-full-length")
				defer os.RemoveAll(path)

				b := []byte{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12}

				cf, err := openCircularFile(path, false, int64(len(b)))
				require.NoError(t, err)
				defer cf.Close()

				require.NotNil(t, cf)

				n, err := cf.Write(b)
				require.NoError(t, err)
				require.Equal(t, len(b), n)
				require.Equal(t, int64(len(b)), cf.len())
				require.True(t, cf.Full)

				bufOut := make([]byte, len(b))
				n, err = cf.Read(bufOut)
				require.NoError(t, err)
				require.Equal(t, len(bufOut), n)
				require.Equal(t, int64(0), cf.Start)
				require.Equal(t, int64(0), cf.readBytesLeft())
			},
		},
		{
			desc: "Reading when empty gives EOF",
			testFunc: func(t *testing.T) {
				t.Parallel()
				path := randomFilePath("ring-buffer-read-empty")
				defer os.RemoveAll(path)

				cf, err := openCircularFile(path, false, int64(12))
				require.NoError(t, err)
				defer cf.Close()

				require.NotNil(t, cf)

				bufOut := make([]byte, 12)
				n, err := cf.Read(bufOut)
				require.ErrorIs(t, err, io.EOF)
				require.Equal(t, 0, n)
				require.Equal(t, int64(0), cf.Start)
			},
		},
		{
			desc: "Reading over boundary is valid",
			testFunc: func(t *testing.T) {
				t.Parallel()
				path := randomFilePath("ring-buffer-read-boundary")
				defer os.RemoveAll(path)

				b := []byte{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12}

				cf, err := openCircularFile(path, false, int64(len(b)))
				require.NoError(t, err)
				defer cf.Close()

				require.NotNil(t, cf)

				cf.Start = int64(len(b) / 2)
				cf.End = cf.Start
				cf.ReadPtr = cf.Start

				n, err := cf.Write(b)
				require.NoError(t, err)
				require.Equal(t, len(b), n)
				require.Equal(t, int64(len(b)), cf.len())
				require.True(t, cf.Full)

				bufOut := make([]byte, len(b))
				n, err = cf.Read(bufOut)
				require.NoError(t, err)
				require.Equal(t, len(bufOut), n)
				require.Equal(t, b, bufOut)
				require.True(t, cf.Full)
			},
		},
		{
			desc: "Reading over boundary (not full) is valid",
			testFunc: func(t *testing.T) {
				t.Parallel()
				path := randomFilePath("ring-buffer-read-boundary-not-full")
				defer os.RemoveAll(path)

				b := []byte{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12}

				cf, err := openCircularFile(path, false, int64(len(b)+1))
				require.NoError(t, err)
				defer cf.Close()

				require.NotNil(t, cf)

				cf.Start = int64(len(b) / 2)
				cf.End = cf.Start
				cf.ReadPtr = cf.Start

				n, err := cf.Write(b)
				require.NoError(t, err)
				require.Equal(t, len(b), n)
				require.Equal(t, int64(len(b)), cf.len())
				require.False(t, cf.Full)

				bufOut := make([]byte, len(b))
				n, err = cf.Read(bufOut)
				require.NoError(t, err)
				require.Equal(t, len(bufOut), n)
				require.Equal(t, b, bufOut)
			},
		},
		{
			desc: "First read fails",
			testFunc: func(t *testing.T) {
				t.Parallel()
				path := randomFilePath("ring-buffer-read-fails")
				defer os.RemoveAll(path)

				b := make([]byte, 13)

				f := &mocks.FileLike{}
				f.On("Seek", mock.Anything, mock.Anything).Return(int64(0), nil)
				f.On("Read", b).Return(2, io.ErrUnexpectedEOF)

				cf := circularFile{
					Size: int64(len(b)),
					f:    f,
					Full: true,
				}

				n, err := cf.Read(b)
				require.ErrorIs(t, err, io.ErrUnexpectedEOF)
				require.Equal(t, 2, n)
				require.Equal(t, int64(len(b)), cf.len())
				require.Equal(t, int64(0), cf.Start)
				require.True(t, cf.Full)
			},
		},
		{
			desc: "Second read fails",
			testFunc: func(t *testing.T) {
				t.Parallel()
				path := randomFilePath("ring-buffer-read2-fails")
				defer os.RemoveAll(path)

				b := make([]byte, 13)
				firstReadAmount := len(b) - len(b)/2
				firstReadSlice := b[:firstReadAmount]
				secondReadSlice := b[firstReadAmount:]

				f := &mocks.FileLike{}
				f.On("Seek", mock.Anything, mock.Anything).Return(int64(0), nil)
				f.On("Read", firstReadSlice).Return(len(firstReadSlice), nil)
				f.On("Read", secondReadSlice).Return(1, io.ErrUnexpectedEOF)

				cf := circularFile{
					Size: int64(len(b)),
					f:    f,
				}

				cf.Start = int64(len(b) / 2)
				cf.End = cf.Start
				cf.ReadPtr = cf.Start
				cf.Full = true

				n, err := cf.Read(b)
				require.ErrorIs(t, err, io.ErrUnexpectedEOF)
				require.Equal(t, len(firstReadSlice)+1, n)
				require.Equal(t, int64(1), cf.ReadPtr)
				require.True(t, cf.Full)
			},
		},
		{
			desc: "First seek fails",
			testFunc: func(t *testing.T) {
				t.Parallel()
				path := randomFilePath("ring-buffer-read-seek-fails")
				defer os.RemoveAll(path)

				b := make([]byte, 13)

				f := &mocks.FileLike{}
				f.On("Seek", mock.Anything, mock.Anything).Return(int64(0), io.ErrNoProgress)

				cf := circularFile{
					Size: int64(len(b)),
					f:    f,
					Full: true,
				}

				n, err := cf.Read(b)
				require.ErrorIs(t, err, io.ErrNoProgress)
				require.Equal(t, 0, n)
				require.Equal(t, int64(len(b)), cf.len())
				require.Equal(t, int64(0), cf.Start)
				require.True(t, cf.Full)
			},
		},
		{
			desc: "Second seek fails",
			testFunc: func(t *testing.T) {
				t.Parallel()
				path := randomFilePath("ring-buffer-read-seek2-fails")
				defer os.RemoveAll(path)

				b := make([]byte, 13)
				firstReadAmount := len(b) - len(b)/2
				firstReadSlice := b[:firstReadAmount]
				secondReadSlice := b[firstReadAmount:]

				f := &mocks.FileLike{}
				f.On("Seek", int64(len(b)/2), io.SeekStart).Return(int64(0), nil)
				f.On("Seek", int64(0), io.SeekStart).Return(int64(0), io.ErrShortBuffer)
				f.On("Read", firstReadSlice).Return(len(firstReadSlice), nil)
				f.On("Read", secondReadSlice).Return(len(secondReadSlice), nil)

				cf := circularFile{
					Size: int64(len(b)),
					f:    f,
				}

				cf.Start = int64(len(b) / 2)
				cf.End = cf.Start
				cf.ReadPtr = cf.Start
				cf.Full = true

				n, err := cf.Read(b)
				require.ErrorIs(t, err, io.ErrShortBuffer)
				require.Equal(t, len(firstReadSlice), n)
				require.Equal(t, int64(0), cf.ReadPtr)
				require.True(t, cf.Full)
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.desc, tc.testFunc)
	}
}

func TestClose(t *testing.T) {
	testCases := []struct {
		desc     string
		testFunc func(*testing.T)
	}{
		{
			desc: "Closing after open doesn't error",
			testFunc: func(t *testing.T) {
				t.Parallel()
				path := randomFilePath("ring-buffer-close")
				defer os.RemoveAll(path)

				cf, err := openCircularFile(path, false, 1000)
				require.NoError(t, err)
				require.NotNil(t, cf)

				err = cf.Close()
				require.NoError(t, err)
			},
		},
		{
			desc: "Cannot read or write after close",
			testFunc: func(t *testing.T) {
				t.Parallel()

				fileMock := &mocks.FileLike{}
				fileMock.On("Close").Once().Return(nil)

				cf := &circularFile{
					f: fileMock,
				}

				err := cf.Close()
				require.NoError(t, err)

				fileMock.AssertExpectations(t)

				_, err = cf.Read(nil)
				require.ErrorIs(t, err, ErrBufferClosed)

				_, err = cf.Write(nil)
				require.ErrorIs(t, err, ErrBufferClosed)
			},
		},
		{
			desc: "Double close doesn't close file twice",
			testFunc: func(t *testing.T) {
				t.Parallel()

				fileMock := &mocks.FileLike{}
				fileMock.On("Close").Once().Return(nil)

				cf := &circularFile{
					f: fileMock,
				}

				err := cf.Close()
				require.NoError(t, err)

				err = cf.Close()
				require.NoError(t, err)

				fileMock.AssertExpectations(t)
			},
		},
		{
			desc: "File close failure",
			testFunc: func(t *testing.T) {
				t.Parallel()

				fileMock := &mocks.FileLike{}
				fileMock.On("Close").Once().Return(io.ErrNoProgress)

				cf := &circularFile{
					f: fileMock,
				}

				err := cf.Close()
				require.ErrorIs(t, err, io.ErrNoProgress)

				err = cf.Close()
				require.NoError(t, err)

				fileMock.AssertExpectations(t)
			},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.desc, tc.testFunc)
	}
}

func TestDiscard(t *testing.T) {
	testCases := []struct {
		desc     string
		testFunc func(*testing.T)
	}{
		{
			desc: "Discard 0 bytes",
			testFunc: func(t *testing.T) {
				t.Parallel()
				path := randomFilePath("ring-buffer-discard")
				defer os.RemoveAll(path)

				cf, err := openCircularFile(path, false, 1000)
				require.NoError(t, err)
				defer cf.Close()

				require.NotNil(t, cf)
				b := []byte{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12}
				n, err := cf.Write(b)
				require.NoError(t, err)
				require.Equal(t, len(b), n)
				require.Equal(t, int64(len(b)), cf.len())

				cf.discard(0)
				require.False(t, cf.seekedRead)
				require.Equal(t, int64(len(b)), cf.len())

				bufOut := make([]byte, len(b))
				n, err = cf.Read(bufOut)
				require.NoError(t, err)
				require.Equal(t, len(bufOut), n)
				require.Equal(t, b, bufOut)
			},
		},
		{
			desc: "Discard 5 bytes",
			testFunc: func(t *testing.T) {
				t.Parallel()
				path := randomFilePath("ring-buffer-discard-5")
				defer os.RemoveAll(path)

				cf, err := openCircularFile(path, false, 1000)
				require.NoError(t, err)
				defer cf.Close()

				require.NotNil(t, cf)
				b := []byte{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12}
				n, err := cf.Write(b)
				require.NoError(t, err)
				require.Equal(t, len(b), n)
				require.Equal(t, int64(len(b)), cf.len())

				cf.discard(5)
				require.False(t, cf.seekedRead)
				require.Equal(t, int64(len(b)-5), cf.len())

				bufOut := make([]byte, len(b)-5)
				n, err = cf.Read(bufOut)
				require.NoError(t, err)
				require.Equal(t, len(bufOut), n)
				require.Equal(t, b[5:], bufOut)
			},
		},
		{
			desc: "Discard all bytes",
			testFunc: func(t *testing.T) {
				t.Parallel()
				path := randomFilePath("ring-buffer-discard-all")
				defer os.RemoveAll(path)

				cf, err := openCircularFile(path, false, 1000)
				require.NoError(t, err)
				defer cf.Close()

				require.NotNil(t, cf)
				b := []byte{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12}
				n, err := cf.Write(b)
				require.NoError(t, err)
				require.Equal(t, len(b), n)
				require.Equal(t, int64(len(b)), cf.len())

				cf.discard(1000)
				require.False(t, cf.seekedRead)
				require.Equal(t, int64(0), cf.len())
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.desc, tc.testFunc)
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
