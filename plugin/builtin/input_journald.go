// +build linux
// +build cgo

package builtin

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/bluemedora/bplogagent/entry"
	"github.com/bluemedora/bplogagent/plugin"
	"github.com/bluemedora/bplogagent/plugin/helper"
	"github.com/coreos/go-systemd/sdjournal"
	"go.uber.org/zap"
)

func init() {
	plugin.Register("journald_input", &JournaldInputConfig{})
}

type JournaldInputConfig struct {
	helper.BasicPluginConfig `mapstructure:",squash" yaml:",inline"`
	helper.BasicInputConfig  `mapstructure:",squash" yaml:",inline"`
	Directory                *string
	Files                    []string
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

	var journal *sdjournal.Journal
	switch {
	case c.Directory != nil:
		journal, err = sdjournal.NewJournalFromDir(*c.Directory)
	case len(c.Files) > 0:
		journal, err = sdjournal.NewJournalFromFiles(c.Files...)
	default:
		journal, err = sdjournal.NewJournal()
	}
	if err != nil {
		return nil, fmt.Errorf("create journal: %w", err)
	}

	journaldInput := &JournaldInput{
		BasicPlugin: basicPlugin,
		BasicInput:  basicInput,
		journal:     journal,
	}
	return journaldInput, nil
}

type JournaldInput struct {
	helper.BasicPlugin
	helper.BasicInput

	journal *sdjournal.Journal

	cancel context.CancelFunc
	wg     *sync.WaitGroup
}

// Start will start generating log entries.
func (g *JournaldInput) Start() error {
	// TODO protect against multiple starts?
	ctx, cancel := context.WithCancel(context.Background())
	g.cancel = cancel
	g.wg = &sync.WaitGroup{}

	g.wg.Add(1)
	go func() {
		defer g.wg.Done()
		for {
			select {
			case <-ctx.Done():
				return
			default:
			}

			num, err := g.journal.Next()
			if err != nil {
				g.Infow("Failed to advance journal cursor", zap.Error(err))
				return
			}

			if num == 0 {
				signal := make(chan struct{})
				go func() {
					g.journal.Wait(sdjournal.IndefiniteWait)
					close(signal)
				}()
				select {
				case <-signal:
					continue
				case <-ctx.Done():
					return
				}
			}

			jEntry, err := g.journal.GetEntry()
			newRecord := make(map[string]interface{})
			for k, v := range jEntry.Fields {
				newRecord[k] = v
			}
			newEntry := entry.Entry{
				Timestamp: time.Unix(0, int64(jEntry.RealtimeTimestamp)*1000), // from microseconds
				Record:    newRecord,
			}

			err = g.Output.Process(&newEntry)
			if err != nil {
				g.Infow("Failed to process entry: %s", zap.Error(err))
			}
			// TODO checkpointing/offset tracking
		}
	}()

	return nil
}

// Stop will stop generating logs.
func (g *JournaldInput) Stop() error {
	g.cancel()
	g.wg.Wait()
	return nil
}
