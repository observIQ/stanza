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
		InputConfig: helper.NewInputConfig(operatorID, operatorName),
		AzureConfig: azure.AzureConfig{
			PrefetchCount: 1000,
			StartAt:       "end",
		},
	}
}

// EventHubInputConfig is the configuration of a Azure Event Hub input operator.
type EventHubInputConfig struct {
	helper.InputConfig `yaml:",inline"`
	azure.AzureConfig  `yaml:",inline"`
}

// Build will build a Azure Event Hub input operator.
func (c *EventHubInputConfig) Build(buildContext operator.BuildContext) ([]operator.Operator, error) {
	inputOperator, err := c.InputConfig.Build(buildContext)
	if err != nil {
		return nil, err
	}

	if err := c.AzureConfig.Validate(); err != nil {
		return nil, err
	}

	var startAtBegining bool
	switch c.StartAt {
	case "beginning":
		startAtBegining = true
	case "end":
		startAtBegining = false
	default:
		return nil, fmt.Errorf("invalid value '%s' for %s parameter 'start_at'", c.StartAt, operatorName)
	}

	eventHubInput := &EventHubInput{
		EventHub: azure.EventHub{
			Namespace:        c.Namespace,
			Name:             c.Name,
			Group:            c.Group,
			ConnStr:          c.ConnectionString,
			PrefetchCount:    c.PrefetchCount,
			StartAtBeginning: startAtBegining,
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
