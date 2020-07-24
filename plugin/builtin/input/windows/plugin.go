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
		buffer:       NewBuffer(),
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
	bookmark     Bookmark
	subscription Subscription
	buffer       Buffer
	channel      string
	maxReads     int
	startAt      string
	pollInterval plugin.Duration
	offsets      helper.Persister
	cancel       context.CancelFunc
	wg           *sync.WaitGroup
}

// Start will start reading events from a subscription.
func (e *EventLogInput) Start() error {
	ctx, cancel := context.WithCancel(context.Background())
	e.cancel = cancel
	e.wg = &sync.WaitGroup{}

	e.bookmark = NewBookmark()
	offsetXML, err := e.getBookmarkOffset()
	if err != nil {
		return fmt.Errorf("failed to retrieve bookmark offset: %s", err)
	}

	if err := e.bookmark.Open(offsetXML); err != nil {
		return fmt.Errorf("failed to open bookmark: %s", err)
	}

	e.subscription = NewSubscription()
	if err := e.subscription.Open(e.channel, e.startAt, e.bookmark); err != nil {
		return fmt.Errorf("failed to open subscription: %s", err)
	}

	e.wg.Add(1)
	go e.readOnInterval(ctx)
	return nil
}

// Stop will stop reading events from a subscription.
func (e *EventLogInput) Stop() error {
	e.cancel()
	e.wg.Wait()

	if err := e.subscription.Close(); err != nil {
		e.Errorf("Failed to close subscription: %s", err)
	}

	if err := e.bookmark.Close(); err != nil {
		e.Errorf("Failed to close bookmark: %s", err)
	}

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
	events, err := e.subscription.Read(e.maxReads)
	if err != nil {
		e.Errorf("Failed to read events from subscription: %s", err)
		return
	}

	for i, event := range events {
		e.processEvent(ctx, event)
		if len(events) == i+1 {
			e.updateBookmarkOffset(event)
		}
		event.Close()
	}
}

// processEvent will process and send an event retrieved from windows event log.
func (e *EventLogInput) processEvent(ctx context.Context, event Event) {
	simpleEvent, err := event.RenderSimple(e.buffer)
	if err != nil {
		e.Errorf("Failed to render simple event: %s", err)
		return
	}

	publisher := NewPublisher()
	if err := publisher.Open(simpleEvent.Provider.Name); err != nil {
		e.Errorf("Failed to open publisher: %s")
		e.sendEvent(ctx, simpleEvent)
		return
	}
	defer publisher.Close()

	formattedEvent, err := event.RenderFormatted(e.buffer, publisher)
	if err != nil {
		e.Errorf("Failed to render formatted event: %s", err)
		e.sendEvent(ctx, simpleEvent)
		return
	}

	e.sendEvent(ctx, formattedEvent)
}

// sendEvent will send EventXML as an entry to the plugin's output.
func (e *EventLogInput) sendEvent(ctx context.Context, eventXML EventXML) {
	entry := eventXML.ToEntry()
	e.Write(ctx, entry)
}

// getBookmarkXML will get the bookmark xml from the offsets database.
func (e *EventLogInput) getBookmarkOffset() (string, error) {
	if err := e.offsets.Load(); err != nil {
		return "", fmt.Errorf("failed to load offsets database: %s", err)
	}

	bytes := e.offsets.Get(e.channel)
	return string(bytes), nil
}

// updateBookmark will update the bookmark xml and save it in the offsets database.
func (e *EventLogInput) updateBookmarkOffset(event Event) {
	if err := e.bookmark.Update(event); err != nil {
		e.Errorf("Failed to update bookmark from event: %s", err)
		return
	}

	bookmarkXML, err := e.bookmark.Render(e.buffer)
	if err != nil {
		e.Errorf("Failed to render bookmark xml: %s", err)
		return
	}

	e.Errorf("Saving bookmark xml: %s", bookmarkXML)

	e.offsets.Set(e.channel, []byte(bookmarkXML))
	if err := e.offsets.Sync(); err != nil {
		e.Errorf("failed to sync offsets database: %s", err)
		return
	}
}
