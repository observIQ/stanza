package buffer

import (
	"bytes"
	"errors"
	"io"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestReadDiskBufferMetadata(t *testing.T) {
	buf := bytes.NewBufferString("\x01\x00\x00\x00\x00\x00\x00\xFF\x01")
	dmd, err := ReadDiskBufferMetadata(buf)

	require.NoError(t, err)
	require.Equal(t, &DiskBufferMetadata{
		Version:     1,
		StartOffset: 65281,
	}, dmd)

	_, err = ReadDiskBufferMetadata(buf)
	require.ErrorIs(nil, err, io.EOF)
}

func TestDiskBufferMetadataWrite(t *testing.T) {
	buf := &bytes.Buffer{}
	dmd := &DiskBufferMetadata{
		Version:     1,
		StartOffset: 65281,
	}

	dmd.Write(buf)
	assert.Equal(t, []byte("\x01\x00\x00\x00\x00\x00\x00\xFF\x01"), buf.Bytes())

	err := dmd.Write(eofWriteSeeker{})
	assert.ErrorIs(t, err, io.EOF)
}

func TestDiskBufferMetadataSync(t *testing.T) {
	buf := &bytes.Buffer{}
	seekableBuf := seekableByteBuffer{
		buf: buf,
	}

	dmd := &DiskBufferMetadata{
		Version:     1,
		StartOffset: 65281,
	}

	dmd.Sync(seekableBuf)
	assert.Equal(t, []byte("\x01\x00\x00\x00\x00\x00\x00\xFF\x01"), buf.Bytes())

	err := dmd.Sync(eofWriteSeeker{})
	assert.ErrorIs(t, err, io.EOF)
}

type eofWriteSeeker struct{}

func (eofWriteSeeker) Write(buf []byte) (int, error) {
	return 0, io.EOF
}

func (eofWriteSeeker) Seek(offset int64, whence int) (int64, error) {
	return 0, nil
}

type seekableByteBuffer struct {
	buf *bytes.Buffer
}

func (s seekableByteBuffer) Write(buf []byte) (int, error) {
	return s.buf.Write(buf)
}

func (s seekableByteBuffer) Seek(offset int64, whence int) (int64, error) {
	if offset == 0 && whence == io.SeekStart {
		s.buf.Reset()
		return 0, nil
	}
	return 0, errors.New("Cannot seek to anywhere other than start")
}
