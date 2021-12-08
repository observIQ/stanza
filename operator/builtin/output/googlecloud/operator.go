package googlecloud

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/observiq/stanza/entry"
	"github.com/observiq/stanza/operator/buffer"
	"github.com/observiq/stanza/operator/flusher"
	"github.com/observiq/stanza/operator/helper"

	"github.com/google/uuid"
	"go.uber.org/zap"
	"google.golang.org/api/option"
	"google.golang.org/genproto/googleapis/api/monitoredres"
	"google.golang.org/genproto/googleapis/logging/v2"
	"google.golang.org/protobuf/proto"
)

// GoogleCloudOutput is an operator that sends logs to google cloud logging.
type GoogleCloudOutput struct {
	helper.OutputOperator
	buffer  buffer.Buffer
	flusher *flusher.Flusher

	client         Client
	clientOptions  []option.ClientOption
	buildClient    ClientBuilder
	requestBuilder RequestBuilder
	projectID      string

	timeout time.Duration
	ctx     context.Context
	cancel  context.CancelFunc
	wg      sync.WaitGroup
}

// Start will start the google cloud operator
func (g *GoogleCloudOutput) Start() error {
	ctx, cancel := context.WithTimeout(context.Background(), g.timeout)
	defer cancel()

	client, err := g.buildClient(ctx, g.clientOptions...)
	if err != nil {
		return fmt.Errorf("failed to create client: %w", err)
	}
	g.client = client
	g.Debugw("Created client", "options", g.clientOptions)

	err = g.testConnection(ctx)
	if err != nil {
		return err
	}
	g.Debug("Completed test connection")

	g.startFlusher()
	g.Debug("Started flusher")

	return nil
}

// Stop will stop the google cloud operator
func (g *GoogleCloudOutput) Stop() error {
	g.Debug("Stopping operator")

	g.cancel()
	g.wg.Wait()
	g.Debug("Wait group completed")

	g.flusher.Stop()
	g.Debug("Stopped flusher")

	if err := g.buffer.Close(); err != nil {
		return fmt.Errorf("failed to close buffer: %w", err)
	}
	g.Debug("Closed buffer")

	switch g.client {
	case nil:
		g.Debug("Skipping client close")
	default:
		if err := g.client.Close(); err != nil {
			return fmt.Errorf("failed to close client: %w", err)
		}
	}

	g.Debug("Stopped operator")
	return nil
}

// Process adds an incoming entry to the buffer
func (g *GoogleCloudOutput) Process(ctx context.Context, e *entry.Entry) error {
	return g.buffer.Add(ctx, e)
}

// testConnection will attempt to send an entry to google cloud logging
func (g *GoogleCloudOutput) testConnection(ctx context.Context) error {
	request := g.createTestRequest()
	g.Debugw("Sending test connection request", "request", request)
	if _, err := g.client.WriteLogEntries(ctx, request); err != nil {
		return fmt.Errorf("test connection failed: %w", err)
	}

	return nil
}

// createTestRequest creates a test request for testing permissions
func (g *GoogleCloudOutput) createTestRequest() *logging.WriteLogEntriesRequest {
	entry := &logging.LogEntry{}
	entry.Payload = &logging.LogEntry_TextPayload{TextPayload: "Test Connection"}
	resource := &monitoredres.MonitoredResource{
		Type: "global",
		Labels: map[string]string{
			"project_id": g.projectID,
		},
	}

	return &logging.WriteLogEntriesRequest{
		LogName:  createLogName(g.projectID, "default"),
		Entries:  []*logging.LogEntry{entry},
		Resource: resource,
		DryRun:   true,
	}
}

// startFlusher will start flushing entries in a separate goroutine
func (g *GoogleCloudOutput) startFlusher() {
	g.Debug("Starting flusher")
	g.wg.Add(1)

	go func() {
		defer g.wg.Done()
		defer g.Debug("Flusher stopped")

		for {
			select {
			case <-g.ctx.Done():
				g.Debug("Context completed while flushing")
				return
			default:
			}

			err := g.flushChunk(g.ctx)
			if err != nil {
				g.Errorw("Failed to flush from buffer", zap.Error(err))
			}
		}
	}()
}

// flushChunk flushes a chunk of entries from the buffer
func (g *GoogleCloudOutput) flushChunk(ctx context.Context) error {
	entries, clearer, err := g.buffer.ReadChunk(ctx)
	if err != nil {
		return fmt.Errorf("failed to read entries from buffer: %w", err)
	}

	chunkID := uuid.New()
	g.Debugw("Read entries from buffer", "entries", len(entries), "chunk_id", chunkID)

	requests := g.requestBuilder.Build(entries)
	g.Debugw("Created write requests", "requests", len(requests), "chunk_id", chunkID)

	flushFunc := func(ctx context.Context) error {
		err := g.send(ctx, requests)
		if err != nil {
			g.Debugw("Failed to send requests", "chunk_id", chunkID, zap.Error(err))
			return err
		}

		g.Debugw("Marking entries as flushed", "chunk_id", chunkID)
		return clearer.MarkAllAsFlushed()
	}

	g.flusher.Do(flushFunc)
	g.Debugw("Submitted requests to the flusher", "requests", len(requests))

	return nil
}

// send will send requests with the operator's client
func (g *GoogleCloudOutput) send(ctx context.Context, requests []*logging.WriteLogEntriesRequest) error {
	for _, request := range requests {
		g.Debugw("Sending write request", "total_entries", len(request.Entries), "request_size", proto.Size(request))
		_, err := g.client.WriteLogEntries(ctx, request)
		if err != nil {
			return fmt.Errorf("failed to send write request: %w", err)
		}
	}

	return nil
}
