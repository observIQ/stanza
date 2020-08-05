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
	"go.uber.org/zap"
	"golang.org/x/text/encoding"
	"golang.org/x/text/transform"
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

	scanner := NewPositionalScanner(file, maxLogSize, startOffset, splitFunc)

	// Make a large, reusable buffer for transforming
	decoder := encoding.NewDecoder()
	decodeBuffer := make([]byte, 16384)

	fileName := filepath.Base(file.Name())
	emit := func(msgBuf []byte) {
		decoder.Reset()
		var nDst int
		for {
			nDst, _, err = decoder.Transform(decodeBuffer, msgBuf, true)
			if err != nil && err == transform.ErrShortDst {
				decodeBuffer = make([]byte, len(decodeBuffer)*2)
				continue
			} else if err != nil {
				inputOperator.Errorw("failed to transform encoding", zap.Error(err))
				return
			}
			break
		}

		e := inputOperator.NewEntry(string(decodeBuffer[:nDst]))
		e.Set(filePathField, path)
		e.Set(fileNameField, fileName)
		inputOperator.Write(ctx, e)
	}

	// Iterate over the tokenized file, emitting entries as we go
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
			} else if err != nil {
				return errors.Wrap(err, "scanner error")
			}
			break
		}

		emit(scanner.Bytes())
		messenger.SetOffset(scanner.Pos())
	}

	// If we're not at the end of the file, and we haven't
	// advanced since last cycle, read the rest of the file as an entry
	if scanner.Pos() < stat.Size() && scanner.Pos() == startOffset && lastSeenFileSize == stat.Size() {
		_, err := file.Seek(scanner.Pos(), 0)
		if err != nil {
			return errors.Wrap(err, "seeking for trailing entry")
		}

		msgBuf := make([]byte, stat.Size()-scanner.Pos())
		n, err := file.Read(msgBuf)
		if err != nil {
			return errors.Wrap(err, "reading trailing entry")
		}
		emit(msgBuf[:n])
		messenger.SetOffset(scanner.Pos() + int64(n))
	}

	return nil
}
