package file

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/observiq/carbon/entry"
	"github.com/observiq/carbon/errors"
	"github.com/observiq/carbon/plugin/helper"
	"go.uber.org/zap"
	"golang.org/x/text/encoding"
)

// ReadToEnd will read entries from a file and send them to the outputs of an input plugin
func ReadToEnd(
	ctx context.Context,
	path string,
	startOffset int64,
	lastSeenFileSize int64,
	messenger fileUpdateMessenger,
	splitFunc bufio.SplitFunc,
	filePathField entry.Field,
	fileNameField entry.Field,
	inputPlugin helper.InputPlugin,
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

	decoder := encoding.NewDecoder()

	// Make a large, reusable buffer for transforming
	decodeBuffer := make([]byte, 16384)

	// If we're not at the end of the file, and we haven't
	// advanced since last cycle, read the rest of the file as an entry
	defer func() {
		if pos < stat.Size() && pos == startOffset && lastSeenFileSize == stat.Size() {
			readRemaining(ctx, file, pos, stat.Size(), messenger, inputPlugin, filePathField, fileNameField, decoder, decodeBuffer)
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

		decoder.Reset()
		nDst, _, err := decoder.Transform(decodeBuffer, scanner.Bytes(), true)
		if err != nil {
			return err
		}

		e := inputPlugin.NewEntry(string(decodeBuffer[:nDst]))
		e.Set(filePathField, path)
		e.Set(fileNameField, filepath.Base(file.Name()))
		inputPlugin.Write(ctx, e)
		messenger.SetOffset(pos)
	}
}

// readRemaining will read the remaining characters in a file as a log entry.
func readRemaining(ctx context.Context, file *os.File, filePos int64, fileSize int64, messenger fileUpdateMessenger, inputPlugin helper.InputPlugin, filePathField, fileNameField entry.Field, encoder *encoding.Decoder, decodeBuffer []byte) {
	_, err := file.Seek(filePos, 0)
	if err != nil {
		inputPlugin.Errorf("failed to seek to read last log entry")
		return
	}

	msgBuf := make([]byte, fileSize-filePos)
	n, err := file.Read(msgBuf)
	if err != nil {
		inputPlugin.Errorf("failed to read trailing log")
		return
	}
	encoder.Reset()
	nDst, _, err := encoder.Transform(decodeBuffer, msgBuf, true)
	if err != nil {
		inputPlugin.Errorw("failed to decode trailing log", zap.Error(err))
	}

	e := inputPlugin.NewEntry(string(decodeBuffer[:nDst]))
	e.Set(filePathField, file.Name())
	e.Set(fileNameField, filepath.Base(file.Name()))
	inputPlugin.Write(ctx, e)
	messenger.SetOffset(filePos + int64(n))
}
