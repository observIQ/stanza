package azure

import (
	"encoding/json"
	"fmt"

	"github.com/Azure/azure-event-hubs-go/v3/persist"
	"github.com/observiq/stanza/operator/helper"
)

// Persister implements persist.CheckpointPersister
type Persister struct {
	DB helper.Persister
}

// Write bodys an Azure Event Hub Checkpoint to the Stanza persistence backend
func (p *Persister) Write(namespace, name, consumerGroup, partitionID string, checkpoint persist.Checkpoint) error {
	key := p.persistenceKey(namespace, name, consumerGroup, partitionID)
	value, err := json.Marshal(checkpoint)
	if err != nil {
		return err
	}

	p.DB.Set(key, value)
	return p.DB.Sync()
}

// Read retrieves an Azure Event Hub Checkpoint from the Stanza persistence backend
func (p *Persister) Read(namespace, name, consumerGroup, partitionID string) (persist.Checkpoint, error) {
	key := p.persistenceKey(namespace, name, consumerGroup, partitionID)
	value := p.DB.Get(key)

	if len(value) < 1 {
		return persist.Checkpoint{}, nil
	}

	var checkpoint persist.Checkpoint
	err := json.Unmarshal(value, &checkpoint)
	return checkpoint, err
}

func (p *Persister) persistenceKey(namespace, name, consumerGroup, partitionID string) string {
	x := fmt.Sprintf("%s-%s-%s-%s", namespace, name, consumerGroup, partitionID)
	return x
}
