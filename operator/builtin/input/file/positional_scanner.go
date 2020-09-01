package file

import (
	"bufio"
	"io"
)

// PositionalScanner is a scanner that maintains position
type PositionalScanner struct {
	pos int64
	*bufio.Scanner
}

// NewPositionalScanner creates a new positional scanner
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

// Pos returns the current position of the scanner
func (ps *PositionalScanner) Pos() int64 {
	return ps.pos
}
