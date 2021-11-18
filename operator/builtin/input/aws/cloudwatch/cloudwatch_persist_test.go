package cloudwatch

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/observiq/stanza/v2/operator/helper/persist"
	"github.com/observiq/stanza/v2/testutil"
	"github.com/stretchr/testify/require"
)

func TestPersisterCache(t *testing.T) {
	persister := Persister{
		base: &persist.NoopPersister{},
	}
	persister.Write(context.Background(), "key", int64(1620666055012))
	value, readErr := persister.Read(context.Background(), "key")
	require.NoError(t, readErr)
	require.Equal(t, int64(1620666055012), value)
}

func TestPersisterLoad(t *testing.T) {
	tempDir := testutil.NewTempDir(t)
	bboltPersister, openPersisterErr := persist.NewBBoltPersister(filepath.Join(tempDir, "test.db"))
	require.NoError(t, openPersisterErr)
	persister := Persister{
		base: bboltPersister,
	}
	persister.Write(context.Background(), "key", 1620666055012)

	value, readErr := persister.Read(context.Background(), "key")
	require.NoError(t, readErr)
	require.Equal(t, int64(1620666055012), value)
}
