package googlecloud

import (
	"context"

	api "cloud.google.com/go/logging/apiv2"
	"github.com/googleapis/gax-go/v2"
	"google.golang.org/api/option"
	"google.golang.org/genproto/googleapis/logging/v2"
)

// Client is an interface for writing entries to google cloud logging
type Client interface {
	WriteLogEntries(ctx context.Context, req *logging.WriteLogEntriesRequest, opts ...gax.CallOption) (*logging.WriteLogEntriesResponse, error)
	Close() error
}

// ClientBuilder is a function that builds a client
type ClientBuilder = func(ctx context.Context, opts ...option.ClientOption) (Client, error)

// newClient creates a new client
func newClient(ctx context.Context, opts ...option.ClientOption) (Client, error) {
	return api.NewClient(ctx, opts...)
}
