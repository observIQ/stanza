package googlecloud

import (
	"github.com/observiq/stanza/entry"
	"go.uber.org/zap"

	"google.golang.org/genproto/googleapis/api/monitoredres"
	"google.golang.org/genproto/googleapis/logging/v2"
)

// RequestBuilder is an interface for creating requests to the google logging API
type RequestBuilder interface {
	Build(entries []*entry.Entry) []*logging.WriteLogEntriesRequest
}

// GoogleRequestBuilder builds google cloud logging requests
type GoogleRequestBuilder struct {
	MaxRequestSize int
	ProjectID      string
	EntryBuilder   EntryBuilder
	*zap.SugaredLogger
}

// Build builds a series of write requests from stanza entries
func (g *GoogleRequestBuilder) Build(entries []*entry.Entry) []*logging.WriteLogEntriesRequest {
	currentSize := 0
	protoEntries := []*logging.LogEntry{}
	requests := []*logging.WriteLogEntriesRequest{}

	for _, entry := range entries {
		protoEntry, protoSize, err := g.EntryBuilder.Build(entry)
		if err != nil {
			g.Errorw("Failed to create protobuf entry. Dropping entry", zap.Any("error", err))
			continue
		}

		if currentSize+protoSize > g.MaxRequestSize {
			g.Debugw("Reached request size limit. Creating request.", "size", currentSize)
			requests = append(requests, g.buildRequest(protoEntries))
			protoEntries = []*logging.LogEntry{}
			currentSize = 0
		}

		protoEntries = append(protoEntries, protoEntry)
		currentSize += protoSize
	}

	if len(entries) > 0 {
		g.Debugw("Creating request from remaining entries", "size", currentSize)
		requests = append(requests, g.buildRequest(protoEntries))
	}

	return requests
}

// buildRequest builds a request from the supplied entries
func (g *GoogleRequestBuilder) buildRequest(entries []*logging.LogEntry) *logging.WriteLogEntriesRequest {
	return &logging.WriteLogEntriesRequest{
		LogName:  createLogName(g.ProjectID, "default"),
		Resource: g.defaultResource(),
		Entries:  entries,
	}
}

// defaultResources creates a default global resource from the operator's projectID
func (g *GoogleRequestBuilder) defaultResource() *monitoredres.MonitoredResource {
	return &monitoredres.MonitoredResource{
		Type: "global",
		Labels: map[string]string{
			"project_id": g.ProjectID,
		},
	}
}
