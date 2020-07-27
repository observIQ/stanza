package file

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/observiq/carbon/entry"
	"github.com/observiq/carbon/errors"
	"github.com/observiq/carbon/operator/helper"
	"golang.org/x/text/encoding"
)

// ReadToEnd will read entries from a file and send them to the outputs of an input operator
func ReadToEnd(
	ctx context.Context,
	path string,
	startOffset int64,
	lastSeenFileSize int64,
	messenger fileUpdateMessenger,
	splitFunc bufio.SplitFunc,
	filePathField entry.Field,
	fileNameField entry.Field,
	inputOperator helper.InputOperator,
	maxLogSize int,
	encoding encoding.Encoding,
) error {
	defer messenger.FinishedReading()

	select {
	case <-ctx.Done():
		return nil
	default:
	}

	file, err := os.Open(path)
	if err != nil {
		return err
	}
	defer file.Close()

	stat, err := file.Stat()
	if err != nil {
		return err
	}
	messenger.SetLastSeenFileSize(stat.Size())

	// Start at the beginning if the file has been truncated
	if stat.Size() < startOffset {
		startOffset = 0
		messenger.SetOffset(0)
	}

	_, err = file.Seek(startOffset, 0)
	if err != nil {
		return fmt.Errorf("seek file: %s", err)
	}

	scanner := bufio.NewScanner(file)
	buf := make([]byte, 0, 16384)
	scanner.Buffer(buf, maxLogSize)
	pos := startOffset
	scanFunc := func(data []byte, atEOF bool) (advance int, token []byte, err error) {
		advance, token, err = splitFunc(data, atEOF)
		pos += int64(advance)
		return
	}
	scanner.Split(scanFunc)

	// Make a large, reusable buffer for transforming
	decoder := encoding.NewDecoder()
	decodeBuffer := make([]byte, 16384)

	emit := func(msgBuf []byte) {
		decoder.Reset()
		nDst, _, err := decoder.Transform(decodeBuffer, msgBuf, true)
		if err != nil {
			panic(err)
		}

		e := inputOperator.NewEntry(string(decodeBuffer[:nDst]))
		e.Set(filePathField, path)
		e.Set(fileNameField, filepath.Base(file.Name()))
		inputOperator.Write(ctx, e)
	}

	// If we're not at the end of the file, and we haven't
	// advanced since last cycle, read the rest of the file as an entry
	defer func() {
		if pos < stat.Size() && pos == startOffset && lastSeenFileSize == stat.Size() {
			_, err := file.Seek(pos, 0)
			if err != nil {
				inputOperator.Errorf("failed to seek to read last log entry")
				return
			}

			msgBuf := make([]byte, stat.Size()-pos)
			n, err := file.Read(msgBuf)
			if err != nil {
				inputOperator.Errorf("failed to read trailing log")
				return
			}
			emit(msgBuf[:n])
			messenger.SetOffset(pos + int64(n))
		}
	}()

	for {
		select {
		case <-ctx.Done():
			return nil
		default:
		}

		ok := scanner.Scan()
		if !ok {
			if err := scanner.Err(); err == bufio.ErrTooLong {
				return errors.NewError("log entry too large", "increase max_log_size or ensure that multiline regex patterns terminate")
			}
			return scanner.Err()
		}

		emit(scanner.Bytes())
		messenger.SetOffset(pos)
	}
}
