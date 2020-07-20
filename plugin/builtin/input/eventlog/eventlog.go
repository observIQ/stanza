// +build windows

package eventlog

import (
	"context"
	"fmt"
	"sync"

	"github.com/observiq/carbon/plugin"
	"github.com/observiq/carbon/plugin/helper"
	"golang.org/x/text/encoding/unicode"
)

func init() {
	plugin.Register("windows_event_input", &WindowsEventConfig{})
}

// WindowsEventConfig is the configuration of a windows event input plugin.
type WindowsEventConfig struct {
	helper.InputConfig `yaml:",inline"`
	Channel            string `json:"channel"           yaml:"channel"`
}

// Build will build a windows event input plugin.
func (c *WindowsEventConfig) Build(context plugin.BuildContext) (plugin.Plugin, error) {
	inputPlugin, err := c.InputConfig.Build(context)
	if err != nil {
		return nil, err
	}

	windowsEventInput := &WindowsEventInput{
		InputPlugin: inputPlugin,
		channel:     c.Channel,
	}
	return windowsEventInput, nil
}

// WindowsEventInput is a plugin that monitors windows event log
type WindowsEventInput struct {
	helper.InputPlugin
	channel string
	cancel  context.CancelFunc
	wg      *sync.WaitGroup
}

// Start will start generating log entries.
func (w *WindowsEventInput) Start() error {
	ctx, cancel := context.WithCancel(context.Background())
	w.cancel = cancel
	w.wg = &sync.WaitGroup{}

	subscription, err := Subscribe(w.channel, 0)
	if err != nil {
		return fmt.Errorf("failed to subscribe to %s: %s", w.channel, err)
	}

	w.wg.Add(1)
	go func() {
		defer w.wg.Done()
		for {
			select {
			case <-ctx.Done():
				return
			default:
			}

			records, err := ReadEvents(subscription, 100)
			if err != nil {
				w.Errorf("Encountered an error reading events: %s", err)
			}

			for _, record := range records {
				xml, _ := unicode.UTF16(unicode.LittleEndian, unicode.UseBOM).NewDecoder().Bytes(record)
				entry := w.NewEntry(string(xml))
				w.Write(ctx, entry)
			}
		}
	}()

	return nil
}

// Stop will stop reading logs.
func (w *WindowsEventInput) Stop() error {
	w.cancel()
	w.wg.Wait()
	return nil
}
