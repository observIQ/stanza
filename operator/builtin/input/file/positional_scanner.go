package file

import (
	"bufio"
	"io"
)

type PositionalScanner struct {
	pos int64
	*bufio.Scanner
}

func NewPositionalScanner(r io.Reader, maxLogSize int, startOffset int64, splitFunc bufio.SplitFunc) *PositionalScanner {
	ps := &PositionalScanner{
		pos:     startOffset,
		Scanner: bufio.NewScanner(r),
	}

	buf := make([]byte, 0, 16384)
	ps.Scanner.Buffer(buf, maxLogSize)

	scanFunc := func(data []byte, atEOF bool) (advance int, token []byte, err error) {
		advance, token, err = splitFunc(data, atEOF)
		ps.pos += int64(advance)
		return
	}
	ps.Scanner.Split(scanFunc)
	return ps
}

func (ps *PositionalScanner) Pos() int64 {
	return ps.pos
}
