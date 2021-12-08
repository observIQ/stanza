package googlecloud

import (
	"context"
	"testing"

	logging "cloud.google.com/go/logging/apiv2"
	"github.com/stretchr/testify/require"
)

func TestNewClient(t *testing.T) {
	client, err := newClient(context.Background())
	require.NoError(t, err)

	_, ok := client.(*logging.Client)
	require.True(t, ok)
}
