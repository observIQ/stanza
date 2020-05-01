// +build linux

package builtin

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

	"github.com/bluemedora/bplogagent/entry"
	"github.com/bluemedora/bplogagent/plugin"
	"github.com/bluemedora/bplogagent/plugin/helper"
	jsoniter "github.com/json-iterator/go"
	"go.uber.org/zap"
)

func init() {
	plugin.Register("journald_input", &JournaldInputConfig{})
}

type JournaldInputConfig struct {
	helper.BasicPluginConfig `mapstructure:",squash" yaml:",inline"`
	helper.BasicInputConfig  `mapstructure:",squash" yaml:",inline"`
	Directory                *string  `mapstructure:"directory"`
	Files                    []string `mapstructure:"files"`
}

func (c JournaldInputConfig) Build(context plugin.BuildContext) (plugin.Plugin, error) {
	basicPlugin, err := c.BasicPluginConfig.Build(context.Logger)
	if err != nil {
		return nil, err
	}

	basicInput, err := c.BasicInputConfig.Build()
	if err != nil {
		return nil, err
	}

	args := make([]string, 0, 10)

	// Export logs in UTC time
	args = append(args, "--utc")

	// Export logs as JSON
	args = append(args, "--output=json")

	// Continue watching logs until cancelled
	args = append(args, "--follow")

	switch {
	case c.Directory != nil:
		args = append(args, "--directory", *c.Directory)
	case len(c.Files) > 0:
		for _, file := range c.Files {
			args = append(args, "--file", file)
		}
	}

	journaldInput := &JournaldInput{
		BasicPlugin: basicPlugin,
		BasicInput:  basicInput,
		binary:      "journalctl",
		args:        args,
		json:        jsoniter.ConfigFastest,
	}
	return journaldInput, nil
}

type JournaldInput struct {
	helper.BasicPlugin
	helper.BasicInput

	binary string
	args   []string

	json   jsoniter.API
	cancel context.CancelFunc
	wg     *sync.WaitGroup
}

// Start will start generating log entries.
func (plugin *JournaldInput) Start() error {
	ctx, cancel := context.WithCancel(context.Background())
	plugin.cancel = cancel
	plugin.wg = &sync.WaitGroup{}

	plugin.Debugw("Starting journald", "args", plugin.args)
	cmd := exec.Command(plugin.binary, plugin.args...)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("failed to get journalctl stdout: %s", err)
	}

	err = cmd.Start()
	if err != nil {
		return fmt.Errorf("start journalctl: %s", err)
	}

	plugin.wg.Add(1)
	go func() {
		defer plugin.wg.Done()
		<-ctx.Done()
		_ = cmd.Process.Signal(os.Interrupt)
		_, _ = cmd.Process.Wait()
	}()

	plugin.wg.Add(1)
	go func() {
		defer plugin.wg.Done()

		stdoutBuf := bufio.NewReader(stdout)

		for {
			line, err := stdoutBuf.ReadBytes('\n')
			if err != nil {
				if err != io.EOF {
					plugin.Errorw("Received error reading from journalctl stdout", zap.Error(err))
				}
				return
			}

			// TODO #172624646 use cursor for offset tracking
			entry, _, err := plugin.parseJournalEntry(line)
			if err != nil {
				plugin.Warnw("Failed to parse journal entry", zap.Error(err))
				continue
			}

			err = plugin.Output.Process(entry)
			if err != nil {
				plugin.Infow("Failed to process entry: %s", zap.Error(err))
			}
		}
	}()

	return nil
}

func (plugin *JournaldInput) parseJournalEntry(line []byte) (*entry.Entry, string, error) {
	var record map[string]interface{}
	err := plugin.json.Unmarshal(line, &record)
	if err != nil {
		return nil, "", err
	}

	timestampInterface, ok := record["__REALTIME_TIMESTAMP"]
	if !ok {
		return nil, "", errors.New("journald record missing __REALTIME_TIMESTAMP field")
	}

	timestampString, ok := timestampInterface.(string)
	if !ok {
		return nil, "", errors.New("timestamp is not a string")
	}

	timestampInt, err := strconv.ParseInt(timestampString, 10, 64)
	if err != nil {
		return nil, "", fmt.Errorf("parse timestamp: %s", err)
	}

	delete(record, "__REALTIME_TIMESTAMP")

	cursorInterface, ok := record["__CURSOR"]
	if !ok {
		return nil, "", errors.New("journald record missing __CURSOR field")
	}

	cursor, ok := cursorInterface.(string)
	if !ok {
		return nil, "", errors.New("cursor is not a string")
	}

	entry := &entry.Entry{
		Timestamp: time.Unix(0, timestampInt*1000), // in microseconds
		Record:    record,
	}

	return entry, cursor, nil
}

// Stop will stop generating logs.
func (g *JournaldInput) Stop() error {
	g.cancel()
	g.wg.Wait()
	return nil
}
