// +build linux

package journald

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strconv"
	"sync"
	"time"

	jsoniter "github.com/json-iterator/go"
	"github.com/observiq/stanza/entry"
	"github.com/observiq/stanza/operator"
	"github.com/observiq/stanza/operator/helper"
	"go.uber.org/zap"
)

func init() {
	operator.Register("journald_input", func() operator.Builder { return NewJournaldInputConfig("") })
}

func NewJournaldInputConfig(operatorID string) *JournaldInputConfig {
	return &JournaldInputConfig{
		InputConfig:  helper.NewInputConfig(operatorID, "journald_input"),
		StartAt:      "end",
		PollInterval: helper.Duration{Duration: 200 * time.Millisecond},
	}
}

// JournaldInputConfig is the configuration of a journald input operator
type JournaldInputConfig struct {
	helper.InputConfig `yaml:",inline"`

	Directory    *string         `json:"directory,omitempty"     yaml:"directory,omitempty"`
	Files        []string        `json:"files,omitempty"         yaml:"files,omitempty"`
	StartAt      string          `json:"start_at,omitempty"      yaml:"start_at,omitempty"`
	PollInterval helper.Duration `json:"poll_interval,omitempty" yaml:"poll_interval,omitempty"`
}

// Build will build a journald input operator from the supplied configuration
func (c JournaldInputConfig) Build(buildContext operator.BuildContext) ([]operator.Operator, error) {
	inputOperator, err := c.InputConfig.Build(buildContext)
	if err != nil {
		return nil, err
	}

	args := make([]string, 0, 10)

	// Export logs in UTC time
	args = append(args, "--utc")

	// Export logs as JSON
	args = append(args, "--output=json")

	// Give raw json output and then exit the process
	args = append(args, "--no-pager")

	switch c.StartAt {
	case "end":
	case "beginning":
		args = append(args, "--no-tail")
	default:
		return nil, fmt.Errorf("invalid value '%s' for parameter 'start_at'", c.StartAt)
	}

	switch {
	case c.Directory != nil:
		if _, err := os.Stat(*c.Directory); os.IsNotExist(err) {
			return nil, fmt.Errorf("invalid value '%s' for parameter 'directory', directory does not exist: %s", *c.Directory, err)
		}
		args = append(args, "--directory", *c.Directory)
	case len(c.Files) > 0:
		for _, file := range c.Files {
			args = append(args, "--file", file)
		}
	}

	journaldInput := &JournaldInput{
		InputOperator: inputOperator,
		persist:       helper.NewScopedDBPersister(buildContext.Database, c.ID()),
		newCmd: func(ctx context.Context, cursor []byte) cmd {
			if cursor != nil {
				args = append(args, "--after-cursor", string(cursor))
			}
			return exec.CommandContext(ctx, "journalctl", args...) // #nosec - ...
			// journalctl is an executable that is required for this operator to function
		},
		json:         jsoniter.ConfigFastest,
		pollInterval: c.PollInterval.Raw(),
	}
	return []operator.Operator{journaldInput}, nil
}

// JournaldInput is an operator that process logs using journald
type JournaldInput struct {
	helper.InputOperator

	newCmd func(ctx context.Context, cursor []byte) cmd

	pollInterval time.Duration

	persist helper.Persister
	json    jsoniter.API
	cancel  context.CancelFunc
	wg      sync.WaitGroup
}

type cmd interface {
	StdoutPipe() (io.ReadCloser, error)
	Start() error
	Wait() error
}

var lastReadCursorKey = "lastReadCursor"

// Start will start generating log entries.
func (operator *JournaldInput) Start() error {
	ctx, cancel := context.WithCancel(context.Background())
	operator.cancel = cancel

	err := operator.persist.Load()
	if err != nil {
		return err
	}

	operator.startPoller(ctx)
	return nil
}

// startPoller kicks off a goroutine that will poll journald periodically,
// checking if there are new files or new logs in the watched files
func (operator *JournaldInput) startPoller(ctx context.Context) {
	go func() {
		globTicker := time.NewTicker(operator.pollInterval)
		defer globTicker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-globTicker.C:
			}

			if err := operator.poll(ctx); err != nil {
				operator.Errorf("error while polling jouranld: %s", err)
			}
		}
	}()
}

// poll checks all the watched paths for new entries
func (operator *JournaldInput) poll(ctx context.Context) error {
	operator.wg.Add(1)

	defer operator.wg.Done()
	defer operator.syncOffsets()

	// Start from a cursor if there is a saved offset
	cursor := operator.persist.Get(lastReadCursorKey)

	// Start journalctl
	cmd := operator.newCmd(ctx, cursor)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("failed to get journalctl stdout: %s", err)
	}
	defer func() {
		if err := stdout.Close(); err != nil {
			operator.Errorf("error closing stdout stream: %s", err)
		}
		if err := cmd.Wait(); err != nil {
			operator.Errorf("failed to stop journalctl sub process: %s", err)
		}
	}()

	err = cmd.Start()
	if err != nil {
		return fmt.Errorf("start journalctl: %s", err)
	}

	stdoutBuf := bufio.NewReader(stdout)

	for {
		select {
		case <-ctx.Done():
			break
		default:
		}

		line, err := stdoutBuf.ReadBytes('\n')
		if err != nil {
			if err != io.EOF {
				operator.Errorw("Received error reading from journalctl stdout", zap.Error(err))
			}
			// return early when at end of journalctl output
			return nil
		}

		entry, cursor, err := operator.parseJournalEntry(line)
		if err != nil {
			operator.Warnw("Failed to parse journal entry", zap.Error(err))
			continue
		}
		operator.persist.Set(lastReadCursorKey, []byte(cursor))
		operator.Write(ctx, entry)
	}
}

func (operator *JournaldInput) parseJournalEntry(line []byte) (*entry.Entry, string, error) {
	var record map[string]interface{}
	err := operator.json.Unmarshal(line, &record)
	if err != nil {
		return nil, "", err
	}

	timestamp, ok := record["__REALTIME_TIMESTAMP"]
	if !ok {
		return nil, "", errors.New("journald record missing __REALTIME_TIMESTAMP field")
	}

	timestampString, ok := timestamp.(string)
	if !ok {
		return nil, "", errors.New("journald field for timestamp is not a string")
	}

	timestampInt, err := strconv.ParseInt(timestampString, 10, 64)
	if err != nil {
		return nil, "", fmt.Errorf("parse timestamp: %s", err)
	}

	delete(record, "__REALTIME_TIMESTAMP")

	cursor, ok := record["__CURSOR"]
	if !ok {
		return nil, "", errors.New("journald record missing __CURSOR field")
	}

	cursorString, ok := cursor.(string)
	if !ok {
		return nil, "", errors.New("journald field for cursor is not a string")
	}

	entry, err := operator.NewEntry(record)
	if err != nil {
		return nil, "", fmt.Errorf("failed to create entry: %s", err)
	}

	entry.Timestamp = time.Unix(0, timestampInt*1000) // in microseconds
	return entry, cursorString, nil
}

func (operator *JournaldInput) syncOffsets() {
	err := operator.persist.Sync()
	if err != nil {
		operator.Errorw("Failed to sync offsets", zap.Error(err))
	}
}

// Stop will stop generating logs.
func (operator *JournaldInput) Stop() error {
	operator.cancel()
	operator.wg.Wait()
	return nil
}
