package eventhub

import (
	"context"
	"fmt"
	"sync"

	azhub "github.com/Azure/azure-event-hubs-go/v3"
	"github.com/Azure/azure-event-hubs-go/v3/persist"
	"github.com/observiq/stanza/operator"
	"github.com/observiq/stanza/operator/helper"
	"go.uber.org/zap"
)

const operatorName = "azure_eventhub_input"

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
func (c *EventHubInputConfig) Build(context operator.BuildContext) ([]operator.Operator, error) {
	inputOperator, err := c.InputConfig.Build(context)
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
		return nil, fmt.Errorf("invalid value '%d' for %s parameter 'start_at'", c.PrefetchCount, operatorName)
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
		InputOperator: inputOperator,
		namespace:     c.Namespace,
		name:          c.Name,
		group:         c.Group,
		connStr:       c.ConnectionString,
		prefetchCount: c.PrefetchCount,
		startAtEnd:    startAtEnd,
	}
	return []operator.Operator{eventHubInput}, nil
}

// EventHubInput is an operator that reads input from Azure Event Hub.
type EventHubInput struct {
	helper.InputOperator
	cancel context.CancelFunc

	namespace     string
	name          string
	group         string
	connStr       string
	prefetchCount uint32
	startAtEnd    bool

	hub *azhub.Hub
	wg  sync.WaitGroup
}

// Start will start generating log entries.
func (e *EventHubInput) Start() error {
	ctx, cancel := context.WithCancel(context.Background())
	e.cancel = cancel

	// TODO: use stanza's offset database
	fp, err := persist.NewFilePersister("storage")
	if err != nil {
		return err
	}

	hub, err := azhub.NewHubFromConnectionString(e.connStr, azhub.HubWithOffsetPersistence(fp))
	if err != nil {
		return err
	}
	e.hub = hub

	runtimeInfo, err := hub.GetRuntimeInformation(ctx)
	if err != nil {
		return err
	}

	for _, partitionID := range runtimeInfo.PartitionIDs {
		go e.poll(ctx, partitionID, fp, hub)
	}

	return nil
}

// Stop will stop generating logs.
func (e *EventHubInput) Stop() error {
	e.cancel()
	e.wg.Wait()
	if err := e.hub.Close(context.Background()); err != nil {
		return err
	}
	e.Infow(fmt.Sprintf("Closed all connections to Azure Event Hub '%s'", e.name))
	return nil
}

// poll starts polling an Azure Event Hub partition id for new events
func (e *EventHubInput) poll(ctx context.Context, partitionID string, fp *persist.FilePersister, hub *azhub.Hub) error {
	offsetStr := ""
	if e.startAtEnd {
		offset, err := fp.Read(e.namespace, e.name, e.group, partitionID)
		if err != nil {
			// TODO: only log if it is an error we don't expect
			x := fmt.Sprintf("Error while reading offset for partition_id %s, starting at begining", partitionID)
			e.Errorw(x, zap.Error(err))
		} else {
			offsetStr = offset.Offset
		}
	}

	var err error

	// start at begining
	if !e.startAtEnd {
		_, err = hub.Receive(
			ctx, partitionID, e.handleEvent, azhub.ReceiveWithStartingOffset(""),
			azhub.ReceiveWithPrefetchCount(e.prefetchCount))
	}
	// start at end and no offset was found
	if e.startAtEnd && offsetStr == "" {
		_, err = hub.Receive(
			ctx, partitionID, e.handleEvent, azhub.ReceiveWithLatestOffset(),
			azhub.ReceiveWithPrefetchCount(e.prefetchCount))
	}
	// start at end and offset exists
	if e.startAtEnd && offsetStr != "" {
		_, err = hub.Receive(
			ctx, partitionID, e.handleEvent, azhub.ReceiveWithStartingOffset(offsetStr),
			azhub.ReceiveWithPrefetchCount(e.prefetchCount))
	}
	if err != nil {
		return err
	}

	e.Infow(fmt.Sprintf("Successfully connected to Azure Event Hub '%s' partition_id '%s'", e.name, partitionID))
	return nil
}

// handleEvents is the handler for hub.Receive.
func (e *EventHubInput) handleEvent(ctx context.Context, event *azhub.Event) error {
	e.wg.Add(1)
	eventEntry, err := parse(event)
	if err != nil {
		return err
	}
	e.Write(ctx, eventEntry)
	e.wg.Done()
	return nil
}
