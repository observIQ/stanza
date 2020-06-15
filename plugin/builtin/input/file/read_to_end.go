package file

import (
	"bufio"
	"context"
	"fmt"
	"os"

	"github.com/bluemedora/bplogagent/entry"
	"github.com/bluemedora/bplogagent/plugin/helper"
)

func ReadToEnd(ctx context.Context, path string, startOffset int64, messenger fileUpdateMessenger, splitFunc bufio.SplitFunc, pathField *entry.Field, inputPlugin helper.InputPlugin) error {
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

	if stat.Size() < startOffset {
		startOffset = 0
		messenger.SetOffset(0)
	}

	_, err = file.Seek(startOffset, 0)
	if err != nil {
		return fmt.Errorf("seek file: %s", err)
	}

	scanner := bufio.NewScanner(file)
	pos := startOffset
	scanFunc := func(data []byte, atEOF bool) (advance int, token []byte, err error) {
		advance, token, err = splitFunc(data, atEOF)
		pos += int64(advance)
		return
	}
	scanner.Split(scanFunc)

	for {
		select {
		case <-ctx.Done():
			return nil
		default:
		}

		ok := scanner.Scan()
		if !ok {
			return scanner.Err()
		}

		message := scanner.Text()

		entry := entry.New()
		entry.Set(inputPlugin.WriteTo, message)
		if pathField != nil {
			entry.Set(*pathField, path)
		}

		err := inputPlugin.Output.Process(ctx, entry)
		if err != nil {
			return err
		}

		messenger.SetOffset(pos)
	}
}
