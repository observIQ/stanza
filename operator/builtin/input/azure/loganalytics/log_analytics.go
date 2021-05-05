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
