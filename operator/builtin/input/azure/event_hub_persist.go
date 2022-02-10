package azure

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/Azure/azure-event-hubs-go/v3/persist"
	"github.com/open-telemetry/opentelemetry-log-collection/operator"
)

// Persister implements persist.CheckpointPersister
type Persister struct {
	base operator.Persister
}

// Write an Azure Event Hub Checkpoint to the Stanza persistence backend
func (p *Persister) Write(namespace, name, consumerGroup, partitionID string, checkpoint persist.Checkpoint) error {
	key := p.persistenceKey(namespace, name, consumerGroup, partitionID)
	value, err := json.Marshal(checkpoint)
	if err != nil {
		return err
	}

	// TODO Revisit this as it's not great using background context but the azure interface won't allow us to pass one in.
	return p.base.Set(context.Background(), key, value)
}

// Read retrieves an Azure Event Hub Checkpoint from the Stanza persistence backend
func (p *Persister) Read(namespace, name, consumerGroup, partitionID string) (persist.Checkpoint, error) {
	var checkpoint persist.Checkpoint

	key := p.persistenceKey(namespace, name, consumerGroup, partitionID)
	// TODO Revisit this as it's not great using background context but the azure interface won't allow us to pass one in.
	value, err := p.base.Get(context.Background(), key)
	if err != nil {
		return checkpoint, err
	}

	if len(value) < 1 {
		return checkpoint, nil
	}

	err = json.Unmarshal(value, &checkpoint)
	return checkpoint, err
}

func (p *Persister) persistenceKey(namespace, name, consumerGroup, partitionID string) string {
	x := fmt.Sprintf("%s-%s-%s-%s", namespace, name, consumerGroup, partitionID)
	return x
}
