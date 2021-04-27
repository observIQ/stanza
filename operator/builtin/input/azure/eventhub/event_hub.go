package eventhub

import (
	"context"
	"fmt"

	"github.com/observiq/stanza/operator"
	"github.com/observiq/stanza/operator/builtin/input/azure"
	"github.com/observiq/stanza/operator/helper"
)

const operatorName = "azure_event_hub_input"

func init() {
	operator.Register(operatorName, func() operator.Builder { return NewEventHubConfig("") })
}

// NewEventHubConfig creates a new Azure Event Hub input config with default values
func NewEventHubConfig(operatorID string) *EventHubInputConfig {
	return &EventHubInputConfig{
		InputConfig:   helper.NewInputConfig(operatorID, operatorName),
		PrefetchCount: 1000,
		StartAt:       "end",
	}
}

// EventHubInputConfig is the configuration of a Azure Event Hub input operator.
type EventHubInputConfig struct {
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

// Build will build a Azure Event Hub input operator.
func (c *EventHubInputConfig) Build(buildContext operator.BuildContext) ([]operator.Operator, error) {
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

	eventHubInput := &EventHubInput{
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
	}
	return []operator.Operator{eventHubInput}, nil
}

// EventHubInput is an operator that reads input from Azure Event Hub.
type EventHubInput struct {
	azure.EventHub
}

// Start will start generating log entries.
func (e *EventHubInput) Start() error {
	e.Handler = e.handleEvent
	return e.StartConsumers(context.Background())
}

// Stop will stop generating logs.
func (e *EventHubInput) Stop() error {
	return e.StopConsumers()
}
