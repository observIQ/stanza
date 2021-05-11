package eventhub

import (
	"context"

	azhub "github.com/Azure/azure-event-hubs-go/v3"
	"github.com/observiq/stanza/operator/builtin/input/azure"
	"go.uber.org/zap"
)

// handleEvent handles an event recieved by an Event Hub consumer.
func (e *EventHubInput) handleEvent(ctx context.Context, event *azhub.Event) error {
	e.WG.Add(1)
	defer e.WG.Done()

	entry, err := e.NewEntry(nil)
	if err != nil {
		e.Errorw("", zap.Error(err))
		return err
	}

	if err := azure.ParseEvent(*event, entry); err != nil {
		e.Errorw("", zap.Error(err))
		return err
	}

	e.Write(ctx, entry)
	return nil
}
