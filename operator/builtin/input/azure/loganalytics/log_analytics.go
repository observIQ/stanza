package loganalytics

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"

	azhub "github.com/Azure/azure-event-hubs-go/v3"
	jsoniter "github.com/json-iterator/go"
	"github.com/observiq/stanza/operator"
	"github.com/observiq/stanza/operator/builtin/input/azure"
	"github.com/observiq/stanza/operator/helper"
	"go.uber.org/zap"
)

const operatorName = "azure_log_analytics_input"

func init() {
	operator.Register(operatorName, func() operator.Builder { return NewLogAnalyticsConfig("") })
}

// NewLogAnalyticsConfig creates a new Azure Log Analytics input config with default values
func NewLogAnalyticsConfig(operatorID string) *LogAnalyticsInputConfig {
	return &LogAnalyticsInputConfig{
		InputConfig:   helper.NewInputConfig(operatorID, operatorName),
		PrefetchCount: 1000,
		StartAt:       "end",
	}
}

// LogAnalyticsInputConfig is the configuration of a Azure Log Analytics input operator.
type LogAnalyticsInputConfig struct {
	helper.InputConfig `yaml:",inline"`

	// required
	Namespace        string `json:"namespace,omitempty"         yaml:"namespace,omitempty"`
	Name             string `json:"name,omitempty"              yaml:"name,omitempty"`
	Group            string `json:"group,omitempty"             yaml:"group,omitempty"`
	ConnectionString string `json:"connection_string,omitempty" yaml:"connection_string,omitempty"`

	// optional
	PrefetchCount uint32 `json:"prefetch_count,omitempty" yaml:"prefetch_count,omitempty"`
	StartAt       string `json:"start_at,omitempty"       yaml:"start_at,omitempty"`
}

// Build will build a Azure Log Analytics input operator.
func (c *LogAnalyticsInputConfig) Build(buildContext operator.BuildContext) ([]operator.Operator, error) {
	inputOperator, err := c.InputConfig.Build(buildContext)
	if err != nil {
		return nil, err
	}

	if c.Namespace == "" {
		return nil, fmt.Errorf("missing required %s parameter 'namespace'", operatorName)
	}

	if c.Name == "" {
		return nil, fmt.Errorf("missing required %s parameter 'name'", operatorName)
	}

	if c.Group == "" {
		return nil, fmt.Errorf("missing required %s parameter 'group'", operatorName)
	}

	if c.ConnectionString == "" {
		return nil, fmt.Errorf("missing required %s parameter 'connection_string'", operatorName)
	}

	if c.PrefetchCount < 1 {
		return nil, fmt.Errorf("invalid value '%d' for %s parameter 'prefetch_count'", c.PrefetchCount, operatorName)
	}

	var startAtEnd bool
	switch c.StartAt {
	case "beginning":
		startAtEnd = false
	case "end":
		startAtEnd = true
	default:
		return nil, fmt.Errorf("invalid value '%s' for %s parameter 'start_at'", c.StartAt, operatorName)
	}

	logAnalyticsInput := &LogAnalyticsInput{
		EventHub: azure.EventHub{
			Namespace:     c.Namespace,
			Name:          c.Name,
			Group:         c.Group,
			ConnStr:       c.ConnectionString,
			PrefetchCount: c.PrefetchCount,
			StartAtEnd:    startAtEnd,
			Persist: &azure.Persister{
				DB: helper.NewScopedDBPersister(buildContext.Database, c.ID()),
			},
			InputOperator: inputOperator,
		},
		json: jsoniter.ConfigFastest,
	}
	return []operator.Operator{logAnalyticsInput}, nil
}

// LogAnalyticsInput is an operator that reads Azure Log Analytics input from Azure Event Hub.
type LogAnalyticsInput struct {
	azure.EventHub
	json jsoniter.API
}

// Start will start generating log entries.
func (l *LogAnalyticsInput) Start() error {
	l.Handler = l.handleBatchedEvents
	return l.StartConsumers(context.Background())
}

// Stop will stop generating logs.
func (l *LogAnalyticsInput) Stop() error {
	return l.StopConsumers()
}

// handleBatchedEvents handles an event recieved by an Event Hub consumer.
func (l *LogAnalyticsInput) handleBatchedEvents(ctx context.Context, event *azhub.Event) error {
	l.WG.Add(1)
	defer l.WG.Done()

	type record struct {
		Records []map[string]interface{} `json:"records"`
	}

	// Create a "base" event by capturing the batch log records from the event's Data field.
	// If Unmarshalling fails, fallback on handling the event as a single log entry.
	records := record{}
	if err := json.Unmarshal(event.Data, &records); err != nil {
		id := event.ID
		if id == "" {
			id = "unknown"
		}
		l.Warnw(fmt.Sprintf("Failed to parse event '%s' as JSON. Expcted key 'records' in event.Data.", string(event.Data)), zap.Error(err))
		l.handleEvent(ctx, *event, nil)
		return nil
	}
	event.Data = nil

	// Create an entry for each log in the batch, using the origonal event's fields
	// as a starting point for each entry
	wg := sync.WaitGroup{}
	max := 10
	gaurd := make(chan struct{}, max)
	for i := 0; i < len(records.Records); i++ {
		r := records.Records[i]
		wg.Add(1)
		gaurd <- struct{}{}
		go func() {
			defer func() {
				wg.Done()
				<-gaurd
			}()
			l.handleEvent(ctx, *event, r)
		}()
	}
	wg.Wait()
	return nil
}

func (l *LogAnalyticsInput) handleEvent(ctx context.Context, event azhub.Event, records map[string]interface{}) {
	e, err := l.NewEntry(nil)
	if err != nil {
		l.Errorw("Failed to parse event as an entry", zap.Error(err))
		return
	}

	if err = l.parse(event, records, e); err != nil {
		l.Errorw("Failed to parse event as an entry", zap.Error(err))
		return
	}
	l.Write(ctx, e)
}
