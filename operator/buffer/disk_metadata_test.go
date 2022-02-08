package buffer

// func TestReadDiskBufferMetadata(t *testing.T) {
// 	buf := bytes.NewBufferString("\x01\x00\x00\x00\x00\x00\x00\xFF\x01")
// 	dmd, err := ReadDiskBufferMetadata(buf)

// 	require.NoError(t, err)
// 	require.Equal(t, &DiskBufferMetadata{
// 		Version:     1,
// 		StartOffset: 65281,
// 	}, dmd)

// 	_, err = ReadDiskBufferMetadata(buf)
// 	require.ErrorIs(nil, err, io.EOF)
// }

// func TestDiskBufferMetadataWrite(t *testing.T) {
// 	buf := &bytes.Buffer{}
// 	dmd := &DiskBufferMetadata{
// 		Version:     1,
// 		StartOffset: 65281,
// 	}

// 	dmd.Write(buf)
// 	assert.Equal(t, []byte("\x01\x00\x00\x00\x00\x00\x00\xFF\x01"), buf.Bytes())

// 	rws := &mockReadWriteSeeker{}
// 	rws.On("Write", mock.Anything).Return(0, io.EOF)

// 	err := dmd.Write(rws)
// 	assert.ErrorIs(t, err, io.EOF)
// }

// func TestDiskBufferMetadataSync(t *testing.T) {
// 	buf := &bytes.Buffer{}

// 	rws := &mockReadWriteSeeker{}
// 	rws.On("Seek", int64(0), io.SeekStart).Return(int64(0), nil)
// 	rws.On("Write", mock.Anything).Run(func(args mock.Arguments) {
// 		buf.Write([]byte("\x01\x00\x00\x00\x00\x00\x00\xFF\x01"))
// 	}).Return(9, nil)

// 	dmd := &DiskBufferMetadata{
// 		Version:     1,
// 		StartOffset: 65281,
// 	}

// 	dmd.Sync(rws)
// 	assert.Equal(t, []byte("\x01\x00\x00\x00\x00\x00\x00\xFF\x01"), buf.Bytes())

// 	rwsSeekFail := &mockReadWriteSeeker{}
// 	rwsSeekFail.On("Seek", int64(0), io.SeekStart).Return(int64(0), io.ErrClosedPipe)

// 	err := dmd.Sync(rwsSeekFail)
// 	assert.ErrorIs(t, err, io.ErrClosedPipe)
// }
