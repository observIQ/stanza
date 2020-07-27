// +build windows

package windows

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/observiq/carbon/operator"
	"github.com/observiq/carbon/operator/helper"
)

func init() {
	operator.Register("windows_eventlog_input", NewDefaultConfig)
}

// EventLogConfig is the configuration of a windows event log operator.
type EventLogConfig struct {
	helper.InputConfig `yaml:",inline"`
	Channel            string            `json:"channel" yaml:"channel"`
	MaxReads           int               `json:"max_reads,omitempty" yaml:"max_reads,omitempty"`
	StartAt            string            `json:"start_at,omitempty" yaml:"start_at,omitempty"`
	PollInterval       operator.Duration `json:"poll_interval,omitempty" yaml:"poll_interval,omitempty"`
}

// Build will build a windows event log operator.
func (c *EventLogConfig) Build(context operator.BuildContext) (operator.Operator, error) {
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
	return eventLogInput, nil
}

// NewDefaultConfig will return an event log config with default values.
func NewDefaultConfig() operator.Builder {
	return &EventLogConfig{
		MaxReads: 100,
		StartAt:  "end",
		PollInterval: operator.Duration{
			Duration: 5 * time.Second,
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
	pollInterval operator.Duration
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

// sendEvent will send EventXML as an entry to the operator's output.
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

	e.offsets.Set(e.channel, []byte(bookmarkXML))
	if err := e.offsets.Sync(); err != nil {
		e.Errorf("failed to sync offsets database: %s", err)
		return
	}
}
