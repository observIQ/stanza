package eventhub

import (
	"encoding/json"
	"fmt"
	"sync"

	"github.com/Azure/azure-event-hubs-go/v3/persist"
	"github.com/observiq/stanza/operator/helper"
)

// Persister implements persist.CheckpointPersister
type Persister struct {
	DB helper.Persister
	mu sync.Mutex
}

// Write records an Azure Event Hub Checkpoint to the Stanza persistence backend
func (p *Persister) Write(namespace, name, consumerGroup, partitionID string, checkpoint persist.Checkpoint) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	key := persistenceKey(namespace, name, consumerGroup, partitionID)
	value, err := json.Marshal(checkpoint)
	if err != nil {
		return err
	}

	p.DB.Set(key, value)

	return nil
}

// Read retrieves an Azure Event Hub Checkpoint from the Stanza persistence backend
func (p *Persister) Read(namespace, name, consumerGroup, partitionID string) (persist.Checkpoint, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	key := persistenceKey(namespace, name, consumerGroup, partitionID)
	value := p.DB.Get(key)

	if len(value) < 1 {
		return persist.Checkpoint{}, nil
	}

	var checkpoint persist.Checkpoint
	err := json.Unmarshal(value, &checkpoint)
	return checkpoint, err
}

func persistenceKey(namespace, name, consumerGroup, partitionID string) string {
	return fmt.Sprintf("%s-%s-%s-%s", namespace, name, consumerGroup, partitionID)
}
