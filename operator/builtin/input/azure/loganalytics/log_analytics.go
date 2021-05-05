package loganalytics

import (
	"context"
	"fmt"

	jsoniter "github.com/json-iterator/go"
	"github.com/observiq/stanza/operator"
	"github.com/observiq/stanza/operator/builtin/input/azure"
	"github.com/observiq/stanza/operator/helper"
)

const operatorName = "azure_log_analytics_input"

func init() {
	operator.Register(operatorName, func() operator.Builder { return NewLogAnalyticsConfig("") })
}

// NewLogAnalyticsConfig creates a new Azure Log Analytics input config with default values
func NewLogAnalyticsConfig(operatorID string) *LogAnalyticsInputConfig {
	return &LogAnalyticsInputConfig{
		InputConfig: helper.NewInputConfig(operatorID, operatorName),
		AzureConfig: azure.AzureConfig{
			PrefetchCount: 1000,
			StartAt:       "end",
		},
	}
}

// LogAnalyticsInputConfig is the configuration of a Azure Log Analytics input operator.
type LogAnalyticsInputConfig struct {
	helper.InputConfig `yaml:",inline"`
	azure.AzureConfig  `yaml:",inline"`
}

// Build will build a Azure Log Analytics input operator.
func (c *LogAnalyticsInputConfig) Build(buildContext operator.BuildContext) ([]operator.Operator, error) {
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

	logAnalyticsInput := &LogAnalyticsInput{
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
