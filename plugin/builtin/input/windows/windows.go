// +build windows

package windows

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/observiq/carbon/plugin"
	"github.com/observiq/carbon/plugin/helper"
)

func init() {
	plugin.Register("windows_eventlog_input", &EventLogConfig{})
}

// EventLogConfig is the configuration of a windows event log plugin.
type EventLogConfig struct {
	helper.InputConfig `yaml:",inline"`
	Channel            string          `json:"channel" yaml:"channel"`
	MaxReads           int             `json:"max_reads,omitempty" yaml:"max_reads,omitempty"`
	StartAt            string          `json:"start_at,omitempty" yaml:"start_at,omitempty"`
	PollInterval       plugin.Duration `json:"poll_interval,omitempty" yaml:"poll_interval,omitempty"`
}

// Build will build a windows event log plugin.
func (c *EventLogConfig) Build(context plugin.BuildContext) (plugin.Plugin, error) {
	inputPlugin, err := c.InputConfig.Build(context)
	if err != nil {
		return nil, err
	}

	if c.Channel == "" {
		return nil, fmt.Errorf("missing required `channel` field")
	}

	maxReads := c.MaxReads
	if maxReads < 1 {
		maxReads = 100
	}

	startAt := c.StartAt
	if startAt == "" {
		startAt = "end"
	}

	pollInterval := c.PollInterval
	if pollInterval.Raw() == 0 {
		pollInterval.Duration = 1 * time.Second
	}

	offsets := helper.NewScopedDBPersister(context.Database, c.ID())

	eventLogInput := &EventLogInput{
		InputPlugin:  inputPlugin,
		channel:      c.Channel,
		maxReads:     maxReads,
		startAt:      startAt,
		pollInterval: pollInterval,
		offsets:      offsets,
	}
	return eventLogInput, nil
}

// EventLogInput is a plugin that creates entries using the windows event log api.
type EventLogInput struct {
	helper.InputPlugin
	channel      string
	maxReads     int
	startAt      string
	pollInterval plugin.Duration
	offsets      helper.Persister
	cancel       context.CancelFunc
	wg           *sync.WaitGroup
	subscription Subscription
}

// Start will start reading events from a subscription.
func (e *EventLogInput) Start() error {
	ctx, cancel := context.WithCancel(context.Background())
	e.cancel = cancel
	e.wg = &sync.WaitGroup{}

	bookmarkXML, err := e.getBookmarkXML()
	if err != nil {
		return fmt.Errorf("failed to retreive bookmark xml: %s", err)
	}

	e.subscription = NewSubscription(e.channel, bookmarkXML, e.maxReads, e.startAt)
	err = e.subscription.Open()
	if err != nil {
		return fmt.Errorf("failed to open subscription: %s", err)
	}

	e.wg.Add(1)
	go e.readOnInterval(ctx)
	return nil
}

// Stop will stop reading events from a subscription.
func (e *EventLogInput) Stop() error {
	err := e.subscription.Close()
	if err != nil {
		e.Errorf("Failed to close subscription: %s", err)
	}

	e.cancel()
	e.wg.Wait()
	return nil
}

// readOnInterval will read events with respect to the polling interval.
func (e *EventLogInput) readOnInterval(ctx context.Context) {
	defer e.wg.Done()

	ticker := time.NewTicker(e.pollInterval.Raw())
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			e.read(ctx)
		}
	}
}

// read will read events from the subscription and update the current offset.
func (e *EventLogInput) read(ctx context.Context) {
	events, err := e.subscription.Read()
	if err != nil {
		e.Errorf("Failed to read events: %s", err)
		return
	}

	for _, event := range events {
		entry := event.ToEntry()
		e.Write(ctx, entry)
	}

	if len(events) > 0 {
		e.saveBookmarkXML()
	}
}

// getBookmarkXML will return the bookmark xml saved in the offsets database.
func (e *EventLogInput) getBookmarkXML() (string, error) {
	err := e.offsets.Load()
	if err != nil {
		return "", fmt.Errorf("failed to load offsets database: %s", err)
	}

	bytes := e.offsets.Get(e.channel)
	return string(bytes), nil
}

// saveBookmarkXML will save the bookmark xml in the offsets database.
func (e *EventLogInput) saveBookmarkXML() error {
	xml := e.subscription.BookmarkXML()
	e.offsets.Set(e.channel, []byte(xml))
	if err := e.offsets.Sync(); err != nil {
		return fmt.Errorf("failed to sync offsets database: %s", err)
	}

	return nil
}
