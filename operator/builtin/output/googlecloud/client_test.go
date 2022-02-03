package googlecloud

import (
	"context"
	"testing"

	logging "cloud.google.com/go/logging/apiv2"
	"github.com/stretchr/testify/require"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/option"
)

func TestNewClient(t *testing.T) {
	json := `{"type": "service_account"}`
	credentials, err := google.CredentialsFromJSON(context.Background(), []byte(json), loggingScope)
	require.NoError(t, err)

	options := option.WithCredentials(credentials)
	client, err := newClient(context.Background(), options)
	require.NoError(t, err)

	_, ok := client.(*logging.Client)
	require.True(t, ok)
}
