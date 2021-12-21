package googlecloud

import (
	"github.com/observiq/stanza/entry"
	"go.uber.org/zap"

	"google.golang.org/genproto/googleapis/api/monitoredres"
	"google.golang.org/genproto/googleapis/logging/v2"
	"google.golang.org/protobuf/proto"
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
	protoEntries := []*logging.LogEntry{}
	for _, entry := range entries {
		protoEntry, err := g.EntryBuilder.Build(entry)
		if err != nil {
			g.Errorw("Failed to create protobuf entry. Dropping entry", zap.Any("error", err))
			continue
		}
		protoEntries = append(protoEntries, protoEntry)
	}

	return g.buildRequests(protoEntries)
}

// buildRequests builds a series of requests from the supplied protobuf entries.
// The number of requests created cooresponds to the max request size of the builder.
func (g *GoogleRequestBuilder) buildRequests(entries []*logging.LogEntry) []*logging.WriteLogEntriesRequest {
	request := g.buildRequest(entries)
	size := proto.Size(request)
	if size <= g.MaxRequestSize {
		g.Debugw("Created write request", "size", size, "entries", len(entries))
		return []*logging.WriteLogEntriesRequest{request}
	}

	if len(request.Entries) == 1 {
		g.Errorw("Single entry exceeds max request size. Dropping entry", "size", size)
		return []*logging.WriteLogEntriesRequest{}
	}

	totalEntries := len(request.Entries)
	firstRequest := g.buildRequest([]*logging.LogEntry{})
	firstSize := 0
	index := 0

	for i, entry := range request.Entries {

		firstRequest.Entries = append(firstRequest.Entries, entry)
		firstSize = proto.Size(firstRequest)

		if firstSize > g.MaxRequestSize {
			index = i
			firstRequest.Entries = firstRequest.Entries[0:index]
			break
		}
	}

	secondEntries := request.Entries[index:totalEntries]
	secondRequests := g.buildRequests(secondEntries)

	return append([]*logging.WriteLogEntriesRequest{firstRequest}, secondRequests...)
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
