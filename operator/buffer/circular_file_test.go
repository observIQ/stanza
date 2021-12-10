package buffer

import (
	"io"
	"os"
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

				rb, err := OpenCircularFile(path, false, 1000)
				require.NoError(t, err)
				defer rb.Close()

				require.NotNil(t, rb)
			},
		},
		{
			desc: "Test open sync has no error",
			testFunc: func(t *testing.T) {
				t.Parallel()
				path := randomFilePath("ring-buffer-open-sync")
				defer os.RemoveAll(path)

				rb, err := OpenCircularFile(path, true, 1000)
				require.NoError(t, err)
				defer rb.Close()

				require.NotNil(t, rb)
			},
		},
		{
			desc: "Test directory path gives error",
			testFunc: func(t *testing.T) {
				t.Parallel()
				path := os.TempDir()

				rb, err := OpenCircularFile(path, true, 1000)
				require.Error(t, err)
				require.Nil(t, rb)
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

				rb, err := OpenCircularFile(path, false, 1000)
				require.NoError(t, err)
				defer rb.Close()

				require.NotNil(t, rb)
				b := []byte{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12}
				n, err := rb.Write(b)
				require.NoError(t, err)
				require.Equal(t, len(b), n)
				require.Equal(t, int64(len(b)), rb.Len())
			},
		},
		{
			desc: "Writing twice has no error",
			testFunc: func(t *testing.T) {
				t.Parallel()
				path := randomFilePath("ring-buffer-write-twice")
				defer os.RemoveAll(path)

				rb, err := OpenCircularFile(path, false, 1000)
				require.NoError(t, err)
				defer rb.Close()

				require.NotNil(t, rb)
				b := []byte{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12}

				n, err := rb.Write(b)
				require.NoError(t, err)
				require.Equal(t, len(b), n)
				require.Equal(t, int64(len(b)), rb.Len())

				n, err = rb.Write(b)
				require.NoError(t, err)
				require.Equal(t, len(b), n)
				require.Equal(t, int64(len(b)*2), rb.Len())
			},
		},
		{
			desc: "Writing rb.Len() bytes is valid",
			testFunc: func(t *testing.T) {
				t.Parallel()
				path := randomFilePath("ring-buffer-write-full-length")
				defer os.RemoveAll(path)

				b := []byte{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12}

				rb, err := OpenCircularFile(path, false, int64(len(b)))
				require.NoError(t, err)
				defer rb.Close()

				require.NotNil(t, rb)

				n, err := rb.Write(b)
				require.NoError(t, err)
				require.Equal(t, len(b), n)
				require.Equal(t, int64(len(b)), rb.Len())
				require.True(t, rb.Full)
			},
		},
		{
			desc: "Writing when full gives EOF",
			testFunc: func(t *testing.T) {
				t.Parallel()
				path := randomFilePath("ring-buffer-write-full-length")
				defer os.RemoveAll(path)

				b := []byte{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12}

				rb, err := OpenCircularFile(path, false, int64(len(b)))
				require.NoError(t, err)
				defer rb.Close()

				require.NotNil(t, rb)

				n, err := rb.Write(b)
				require.NoError(t, err)
				require.Equal(t, len(b), n)
				require.Equal(t, int64(len(b)), rb.Len())
				require.True(t, rb.Full)

				n, err = rb.Write(b)
				require.ErrorIs(t, err, io.EOF)
				require.Equal(t, 0, n)
				require.Equal(t, int64(len(b)), rb.Len())
				require.True(t, rb.Full)
			},
		},
		{
			desc: "Writing over boundary is valid",
			testFunc: func(t *testing.T) {
				t.Parallel()
				path := randomFilePath("ring-buffer-write-boundary")
				defer os.RemoveAll(path)

				b := []byte{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12}

				rb, err := OpenCircularFile(path, false, int64(len(b)))
				require.NoError(t, err)
				defer rb.Close()

				require.NotNil(t, rb)

				rb.Start = int64(len(b) / 2)
				rb.End = rb.Start
				rb.ReadPtr = rb.Start

				n, err := rb.Write(b)
				require.NoError(t, err)
				require.Equal(t, len(b), n)
				require.Equal(t, int64(len(b)), rb.Len())
				require.True(t, rb.Full)
			},
		},
		{
			desc: "Trying to write over buffer returns EOF",
			testFunc: func(t *testing.T) {
				t.Parallel()
				path := randomFilePath("ring-buffer-write-overflow")
				defer os.RemoveAll(path)

				b := []byte{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12}

				rb, err := OpenCircularFile(path, false, int64(len(b)-1))
				require.NoError(t, err)
				defer rb.Close()

				require.NotNil(t, rb)

				n, err := rb.Write(b)
				require.ErrorIs(t, err, io.EOF)
				require.Equal(t, len(b)-1, n)
				require.Equal(t, int64(len(b)-1), rb.Len())
				require.True(t, rb.Full)
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

				rb := CircularFile{
					Size: int64(len(b)),
					f:    f,
				}

				n, err := rb.Write(b)
				require.ErrorIs(t, err, io.ErrUnexpectedEOF)
				require.Equal(t, 2, n)
				require.Equal(t, int64(2), rb.Len())
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

				rb := CircularFile{
					Size: int64(len(b)),
					f:    f,
				}

				rb.Start = int64(len(b) / 2)
				rb.End = rb.Start
				rb.ReadPtr = rb.Start

				n, err := rb.Write(b)
				require.ErrorIs(t, err, io.ErrShortWrite)
				require.Equal(t, len(firstWriteSlice)+1, n)
				require.Equal(t, int64(len(firstWriteSlice))+1, rb.Len())
				require.Equal(t, int64(1), rb.End)
				require.False(t, rb.Full)
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

				rb := CircularFile{
					Size: int64(len(b)),
					f:    f,
				}

				n, err := rb.Write(b)
				require.ErrorIs(t, err, io.ErrClosedPipe)
				require.Equal(t, 0, n)
				require.Equal(t, int64(0), rb.Len())
				require.Equal(t, int64(0), rb.End)
				require.False(t, rb.Full)
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

				rb := CircularFile{
					Size: int64(len(b)),
					f:    f,
				}

				rb.Start = int64(len(b) / 2)
				rb.End = rb.Start
				rb.ReadPtr = rb.Start

				n, err := rb.Write(b)
				require.ErrorIs(t, err, io.ErrClosedPipe)
				require.Equal(t, len(firstWriteSlice), n)
				require.Equal(t, int64(len(firstWriteSlice)), rb.Len())
				require.Equal(t, int64(0), rb.End)
				require.False(t, rb.Full)
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

				rb, err := OpenCircularFile(path, false, 1000)
				require.NoError(t, err)
				defer rb.Close()

				require.NotNil(t, rb)
				b := []byte{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12}
				n, err := rb.Write(b)
				require.NoError(t, err)
				require.Equal(t, len(b), n)
				require.Equal(t, int64(len(b)), rb.Len())

				bufOut := make([]byte, len(b))
				n, err = rb.Read(bufOut)
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

				rb, err := OpenCircularFile(path, false, 1000)
				require.NoError(t, err)
				defer rb.Close()

				require.NotNil(t, rb)
				b := []byte{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13}
				n, err := rb.Write(b)
				require.NoError(t, err)
				require.Equal(t, len(b), n)
				require.Equal(t, int64(len(b)), rb.Len())

				bufOut := make([]byte, len(b))
				n, err = rb.Read(bufOut[:7])
				require.NoError(t, err)
				require.Equal(t, 7, n)
				require.Equal(t, b[:7], bufOut[:7])

				n, err = rb.Read(bufOut[7:])
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

				rb, err := OpenCircularFile(path, false, int64(len(b)))
				require.NoError(t, err)
				defer rb.Close()

				require.NotNil(t, rb)

				n, err := rb.Write(b)
				require.NoError(t, err)
				require.Equal(t, len(b), n)
				require.Equal(t, int64(len(b)), rb.Len())
				require.True(t, rb.Full)

				bufOut := make([]byte, 12)
				n, err = rb.Read(bufOut)
				require.NoError(t, err)
				require.Equal(t, len(bufOut), n)
				require.Equal(t, int64(0), rb.Start)
			},
		},
		{
			desc: "Reading when empty gives EOF",
			testFunc: func(t *testing.T) {
				t.Parallel()
				path := randomFilePath("ring-buffer-read-empty")
				defer os.RemoveAll(path)

				rb, err := OpenCircularFile(path, false, int64(12))
				require.NoError(t, err)
				defer rb.Close()

				require.NotNil(t, rb)

				bufOut := make([]byte, 12)
				n, err := rb.Read(bufOut)
				require.ErrorIs(t, err, io.EOF)
				require.Equal(t, 0, n)
				require.Equal(t, int64(0), rb.Start)
			},
		},
		{
			desc: "Reading over boundary is valid",
			testFunc: func(t *testing.T) {
				t.Parallel()
				path := randomFilePath("ring-buffer-read-boundary")
				defer os.RemoveAll(path)

				b := []byte{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12}

				rb, err := OpenCircularFile(path, false, int64(len(b)))
				require.NoError(t, err)
				defer rb.Close()

				require.NotNil(t, rb)

				rb.Start = int64(len(b) / 2)
				rb.End = rb.Start
				rb.ReadPtr = rb.Start

				n, err := rb.Write(b)
				require.NoError(t, err)
				require.Equal(t, len(b), n)
				require.Equal(t, int64(len(b)), rb.Len())
				require.True(t, rb.Full)

				bufOut := make([]byte, len(b))
				n, err = rb.Read(bufOut)
				require.NoError(t, err)
				require.Equal(t, len(bufOut), n)
				require.Equal(t, b, bufOut)
				require.True(t, rb.Full)
			},
		},
		{
			desc: "Reading over boundary (not full) is valid",
			testFunc: func(t *testing.T) {
				t.Parallel()
				path := randomFilePath("ring-buffer-read-boundary-not-full")
				defer os.RemoveAll(path)

				b := []byte{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12}

				rb, err := OpenCircularFile(path, false, int64(len(b)+1))
				require.NoError(t, err)
				defer rb.Close()

				require.NotNil(t, rb)

				rb.Start = int64(len(b) / 2)
				rb.End = rb.Start
				rb.ReadPtr = rb.Start

				n, err := rb.Write(b)
				require.NoError(t, err)
				require.Equal(t, len(b), n)
				require.Equal(t, int64(len(b)), rb.Len())
				require.False(t, rb.Full)

				bufOut := make([]byte, len(b))
				n, err = rb.Read(bufOut)
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

				rb := CircularFile{
					Size: int64(len(b)),
					f:    f,
					Full: true,
				}

				n, err := rb.Read(b)
				require.ErrorIs(t, err, io.ErrUnexpectedEOF)
				require.Equal(t, 2, n)
				require.Equal(t, int64(len(b)), rb.Len())
				require.Equal(t, int64(0), rb.Start)
				require.True(t, rb.Full)
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

				rb := CircularFile{
					Size: int64(len(b)),
					f:    f,
				}

				rb.Start = int64(len(b) / 2)
				rb.End = rb.Start
				rb.ReadPtr = rb.Start
				rb.Full = true

				n, err := rb.Read(b)
				require.ErrorIs(t, err, io.ErrUnexpectedEOF)
				require.Equal(t, len(firstReadSlice)+1, n)
				require.Equal(t, int64(1), rb.ReadPtr)
				require.True(t, rb.Full)
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

				rb := CircularFile{
					Size: int64(len(b)),
					f:    f,
					Full: true,
				}

				n, err := rb.Read(b)
				require.ErrorIs(t, err, io.ErrNoProgress)
				require.Equal(t, 0, n)
				require.Equal(t, int64(len(b)), rb.Len())
				require.Equal(t, int64(0), rb.Start)
				require.True(t, rb.Full)
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

				rb := CircularFile{
					Size: int64(len(b)),
					f:    f,
				}

				rb.Start = int64(len(b) / 2)
				rb.End = rb.Start
				rb.ReadPtr = rb.Start
				rb.Full = true

				n, err := rb.Read(b)
				require.ErrorIs(t, err, io.ErrShortBuffer)
				require.Equal(t, len(firstReadSlice), n)
				require.Equal(t, int64(0), rb.ReadPtr)
				require.True(t, rb.Full)
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

				rb, err := OpenCircularFile(path, false, 1000)
				require.NoError(t, err)
				require.NotNil(t, rb)

				err = rb.Close()
				require.NoError(t, err)
			},
		},
		{
			desc: "Cannot read or write after close",
			testFunc: func(t *testing.T) {
				t.Parallel()

				fileMock := &mocks.FileLike{}
				fileMock.On("Close").Once().Return(nil)

				rb := &CircularFile{
					f: fileMock,
				}

				err := rb.Close()
				require.NoError(t, err)

				fileMock.AssertExpectations(t)

				_, err = rb.Read(nil)
				require.ErrorIs(t, err, ErrBufferClosed)

				_, err = rb.Write(nil)
				require.ErrorIs(t, err, ErrBufferClosed)
			},
		},
		{
			desc: "Double close doesn't close file twice",
			testFunc: func(t *testing.T) {
				t.Parallel()

				fileMock := &mocks.FileLike{}
				fileMock.On("Close").Once().Return(nil)

				rb := &CircularFile{
					f: fileMock,
				}

				err := rb.Close()
				require.NoError(t, err)

				err = rb.Close()
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

				rb := &CircularFile{
					f: fileMock,
				}

				err := rb.Close()
				require.ErrorIs(t, err, io.ErrNoProgress)

				err = rb.Close()
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

				rb, err := OpenCircularFile(path, false, 1000)
				require.NoError(t, err)
				defer rb.Close()

				require.NotNil(t, rb)
				b := []byte{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12}
				n, err := rb.Write(b)
				require.NoError(t, err)
				require.Equal(t, len(b), n)
				require.Equal(t, int64(len(b)), rb.Len())

				rb.Discard(0)
				require.False(t, rb.seekedRead)
				require.Equal(t, int64(len(b)), rb.Len())

				bufOut := make([]byte, len(b))
				n, err = rb.Read(bufOut)
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

				rb, err := OpenCircularFile(path, false, 1000)
				require.NoError(t, err)
				defer rb.Close()

				require.NotNil(t, rb)
				b := []byte{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12}
				n, err := rb.Write(b)
				require.NoError(t, err)
				require.Equal(t, len(b), n)
				require.Equal(t, int64(len(b)), rb.Len())

				rb.Discard(5)
				require.False(t, rb.seekedRead)
				require.Equal(t, int64(len(b)-5), rb.Len())

				bufOut := make([]byte, len(b)-5)
				n, err = rb.Read(bufOut)
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

				rb, err := OpenCircularFile(path, false, 1000)
				require.NoError(t, err)
				defer rb.Close()

				require.NotNil(t, rb)
				b := []byte{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12}
				n, err := rb.Write(b)
				require.NoError(t, err)
				require.Equal(t, len(b), n)
				require.Equal(t, int64(len(b)), rb.Len())

				rb.Discard(1000)
				require.False(t, rb.seekedRead)
				require.Equal(t, int64(0), rb.Len())
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.desc, tc.testFunc)
	}
}
