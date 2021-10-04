// +build windows

package windows

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/observiq/stanza/operator"
	"github.com/observiq/stanza/operator/helper"
)

func init() {
	operator.Register("windows_eventlog_input", NewDefaultConfig)
}

// EventLogConfig is the configuration of a windows event log operator.
type EventLogConfig struct {
	helper.InputConfig `yaml:",inline"`
	Channel            string          `json:"channel" yaml:"channel"`
	MaxReads           int             `json:"max_reads,omitempty" yaml:"max_reads,omitempty"`
	StartAt            string          `json:"start_at,omitempty" yaml:"start_at,omitempty"`
	PollInterval       helper.Duration `json:"poll_interval,omitempty" yaml:"poll_interval,omitempty"`
}

// Build will build a windows event log operator.
func (c *EventLogConfig) Build(context operator.BuildContext) ([]operator.Operator, error) {
	inputOperator, err := c.InputConfig.Build(context)
	if err != nil {
		return nil, err
	}

	if c.Channel == "" {
		return nil, fmt.Errorf("missing required `channel` field")
	}

	if c.MaxReads < 1 {
		return nil, fmt.Errorf("the `max_reads` field must be greater than zero")
	}

	if c.StartAt != "end" && c.StartAt != "beginning" {
		return nil, fmt.Errorf("the `start_at` field must be set to `beginning` or `end`")
	}

	offsets := helper.NewScopedDBPersister(context.Database, c.ID())

	eventLogInput := &EventLogInput{
		InputOperator: inputOperator,
		buffer:        NewBuffer(),
		channel:       c.Channel,
		maxReads:      c.MaxReads,
		startAt:       c.StartAt,
		pollInterval:  c.PollInterval,
		offsets:       offsets,
	}
	return []operator.Operator{eventLogInput}, nil
}

// NewDefaultConfig will return an event log config with default values.
func NewDefaultConfig() operator.Builder {
	return &EventLogConfig{
		InputConfig: helper.NewInputConfig("", "windows_eventlog_input"),
		MaxReads:    100,
		StartAt:     "end",
		PollInterval: helper.Duration{
			Duration: 1 * time.Second,
		},
	}
}

// EventLogInput is an operator that creates entries using the windows event log api.
type EventLogInput struct {
	helper.InputOperator
	bookmark     Bookmark
	subscription Subscription
	buffer       Buffer
	channel      string
	maxReads     int
	startAt      string
	pollInterval helper.Duration
	offsets      helper.Persister
	cancel       context.CancelFunc
	wg           sync.WaitGroup
}

// Start will start reading events from a subscription.
func (e *EventLogInput) Start() error {
	ctx, cancel := context.WithCancel(context.Background())
	e.cancel = cancel

	e.bookmark = NewBookmark()
	offsetXML, err := e.getBookmarkOffset()
	if err != nil {
		return fmt.Errorf("failed to retrieve bookmark offset: %s", err)
	}

	if offsetXML != "" {
		if err := e.bookmark.Open(offsetXML); err != nil {
			e.Errorf("Failed to open bookmark, continuing without previous bookmark: %s", err)
			e.offsets.Set(e.channel, []byte{})
			if err := e.offsets.Sync(); err != nil {
				return fmt.Errorf("Could not sync empty bookmark to offsets database: %s", err)
			}
		}
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
		return fmt.Errorf("failed to close subscription: %s", err)
	}

	if err := e.bookmark.Close(); err != nil {
		return fmt.Errorf("failed to close bookmark: %s", err)
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
			e.readToEnd(ctx)
		}
	}
}

// readToEnd will read events from the subscription until it reaches the end of the channel.
func (e *EventLogInput) readToEnd(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		default:
			if count := e.read(ctx); count == 0 {
				return
			}
		}
	}
}

// read will read events from the subscription.
func (e *EventLogInput) read(ctx context.Context) int {
	events, err := e.subscription.Read(e.maxReads)
	if err != nil {
		e.Errorf("Failed to read events from subscription: %s", err)
		return 0
	}

	for i, event := range events {
		e.processEvent(ctx, event)
		if len(events) == i+1 {
			e.updateBookmarkOffset(event)
		}
		event.Close()
	}

	return len(events)
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
		e.Debugf("Failed to open publisher: %s: Submitting entry without further parsing", err)
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

// sendEvent will send EventXML as an entry to the operator's output.
func (e *EventLogInput) sendEvent(ctx context.Context, eventXML EventXML) {
	record := eventXML.parseRecord()
	entry, err := e.NewEntry(record)
	if err != nil {
		e.Errorf("Failed to create entry: %s", err)
		return
	}

	entry.Timestamp = eventXML.parseTimestamp()
	entry.Severity = eventXML.parseSeverity()
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

	e.offsets.Set(e.channel, []byte(bookmarkXML))
	if err := e.offsets.Sync(); err != nil {
		e.Errorf("failed to sync offsets database: %s", err)
		return
	}
}
