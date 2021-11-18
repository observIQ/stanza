package eventhub

import (
	"context"

	"github.com/observiq/stanza/v2/operator"
	"github.com/observiq/stanza/v2/operator/builtin/input/azure"
	"github.com/observiq/stanza/v2/operator/helper"
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
	if err := c.AzureConfig.Build(buildContext, c.InputConfig); err != nil {
		return nil, err
	}

	eventHubInput := &EventHubInput{
		EventHub: azure.EventHub{
			AzureConfig: c.AzureConfig,
			Persist: &azure.Persister{
				DB: helper.NewScopedDBPersister(buildContext.Database, c.ID()),
			},
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
