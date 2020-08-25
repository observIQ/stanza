package disk

import "io"

type CountingWriter struct {
	w io.Writer
	n int
}

func (c *CountingWriter) Write(dst []byte) (int, error) {
	n, err := c.w.Write(dst)
	c.n += n
	return n, err
}

func (c *CountingWriter) BytesWritten() int {
	return c.n
}

func NewCountingWriter(w io.Writer) *CountingWriter {
	return &CountingWriter{
		w: w,
	}
}

type CountingReader struct {
	r io.Reader
	n int
}

func (c *CountingReader) Read(dst []byte) (int, error) {
	n, err := c.r.Read(dst)
	c.n += n
	return n, err
}

func (c *CountingReader) BytesRead() int {
	return c.n
}

func NewCountingReader(r io.Reader) *CountingReader {
	return &CountingReader{
		r: r,
	}
}
