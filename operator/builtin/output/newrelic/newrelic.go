package newrelic

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"sync"
	"time"

	"github.com/observiq/stanza/v2/operator/buffer"
	"github.com/observiq/stanza/v2/operator/flusher"
	"github.com/open-telemetry/opentelemetry-log-collection/entry"
	otelerrors "github.com/open-telemetry/opentelemetry-log-collection/errors"
	"github.com/open-telemetry/opentelemetry-log-collection/operator"
	"github.com/open-telemetry/opentelemetry-log-collection/operator/helper"
	"go.uber.org/zap"
)

func init() {
	operator.Register("newrelic_output", func() operator.Builder { return NewNewRelicOutputConfig("") })
}

// NewNewRelicOutputConfig creates a new relic output config with default values
func NewNewRelicOutputConfig(operatorID string) *NewRelicOutputConfig {
	return &NewRelicOutputConfig{
		OutputConfig:  helper.NewOutputConfig(operatorID, "newrelic_output"),
		BufferConfig:  buffer.NewConfig(),
		FlusherConfig: flusher.NewConfig(),
		BaseURI:       "https://log-api.newrelic.com/log/v1",
		Timeout:       helper.NewDuration(10 * time.Second),
		MessageField:  entry.NewBodyField(),
	}
}

// NewRelicOutputConfig is the configuration of a NewRelicOutput operator
type NewRelicOutputConfig struct {
	helper.OutputConfig `yaml:",inline"`
	BufferConfig        buffer.Config  `json:"buffer" yaml:"buffer"`
	FlusherConfig       flusher.Config `json:"flusher" yaml:"flusher"`

	APIKey       string          `json:"api_key,omitempty"       yaml:"api_key,omitempty"`
	BaseURI      string          `json:"base_uri,omitempty"      yaml:"base_uri,omitempty"`
	LicenseKey   string          `json:"license_key,omitempty"   yaml:"license_key,omitempty"`
	Timeout      helper.Duration `json:"timeout,omitempty"       yaml:"timeout,omitempty"`
	MessageField entry.Field     `json:"message_field,omitempty" yaml:"message_field,omitempty"`
}

// Build will build a new NewRelicOutput
func (c NewRelicOutputConfig) Build(bc operator.BuildContext) ([]operator.Operator, error) {
	outputOperator, err := c.OutputConfig.Build(bc)
	if err != nil {
		return nil, err
	}

	headers, err := c.getHeaders()
	if err != nil {
		return nil, err
	}

	buffer, err := c.BufferConfig.Build()
	if err != nil {
		return nil, err
	}

	url, err := url.Parse(c.BaseURI)
	if err != nil {
		return nil, otelerrors.Wrap(err, "'base_uri' is not a valid URL")
	}

	flusher := c.FlusherConfig.Build(bc.Logger.SugaredLogger)
	ctx, cancel := context.WithCancel(context.Background())

	nro := &NewRelicOutput{
		OutputOperator: outputOperator,
		buffer:         buffer,
		flusher:        flusher,
		client:         NewClient(url, headers),
		timeout:        c.Timeout.Raw(),
		messageField:   c.MessageField,
		ctx:            ctx,
		cancel:         cancel,
	}

	return []operator.Operator{nro}, nil
}

func (c NewRelicOutputConfig) getHeaders() (http.Header, error) {
	headers := http.Header{
		"X-Event-Source":   []string{"logs"},
		"Content-Encoding": []string{"gzip"},
	}

	if c.APIKey == "" && c.LicenseKey == "" {
		return nil, fmt.Errorf("one of 'api_key' or 'license_key' is required")
	} else if c.APIKey != "" {
		headers["X-Insert-Key"] = []string{c.APIKey}
	} else {
		headers["X-License-Key"] = []string{c.LicenseKey}
	}

	return headers, nil
}

// NewRelicOutput is an operator that sends entries to the New Relic Logs platform
type NewRelicOutput struct {
	helper.OutputOperator
	buffer  buffer.Buffer
	flusher *flusher.Flusher

	client       Client
	timeout      time.Duration
	messageField entry.Field

	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup
}

// Start tests the connection to New Relic and begins flushing entries
func (nro *NewRelicOutput) Start(_ operator.Persister) error {
	if err := nro.testConnection(); err != nil {
		return fmt.Errorf("test connection: %s", err)
	}

	nro.wg.Add(1)
	go func() {
		defer nro.wg.Done()
		nro.feedFlusher(nro.ctx)
	}()

	return nil
}

// Stop tells the NewRelicOutput to stop gracefully
func (nro *NewRelicOutput) Stop() error {
	nro.cancel()
	nro.wg.Wait()
	nro.flusher.Stop()
	// TODO deal with buffer Drain
	entries, err := nro.buffer.Close()
	if err != nil {
		return fmt.Errorf("failed to close buffer: %w", err)
	}

	if len(entries) != 0 {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
		defer cancel()

		err = nro.sendEntries(ctx, entries)
		if err != nil {
			return err
		}
	}

	return err
}

// Process adds an entry to the output's buffer
func (nro *NewRelicOutput) Process(ctx context.Context, entry *entry.Entry) error {
	return nro.buffer.Add(ctx, entry)
}

func (nro *NewRelicOutput) sendEntries(ctx context.Context, entries []*entry.Entry) error {
	payload := LogPayloadFromEntries(entries, nro.messageField)
	err := nro.client.SendPayload(ctx, payload)
	if err != nil {
		return fmt.Errorf("Failed to send entries: %s", err)
	}

	return nil
}

func (nro *NewRelicOutput) testConnection() error {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()

	return nro.client.TestConnection(ctx)
}

func (nro *NewRelicOutput) feedFlusher(ctx context.Context) {
	for {
		entries, err := nro.buffer.Read(ctx)
		switch {
		case errors.Is(err, context.Canceled):
			return
		case err != nil:
			nro.flusher.Errorf("Failed to read chunk", zap.Error(err))
			continue
		}

		nro.flusher.Do(ctx, func(flushCtx context.Context) error {
			return nro.sendEntries(flushCtx, entries)
		})
	}
}
