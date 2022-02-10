package persist

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNoopPersisterGet(t *testing.T) {
	persister := &NoopPersister{}

	val, err := persister.Get(context.Background(), "")
	assert.Nil(t, val)
	assert.NoError(t, err)
}

func TestNoopPersisterSet(t *testing.T) {
	persister := &NoopPersister{}
	assert.NoError(t, persister.Set(context.Background(), "", nil))
}

func TestNoopPersisterDelete(t *testing.T) {
	persister := &NoopPersister{}
	assert.NoError(t, persister.Delete(context.Background(), ""))
}
