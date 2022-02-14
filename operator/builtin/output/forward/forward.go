package forward

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io/ioutil"
	"net/http"
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
	operator.Register("forward_output", func() operator.Builder { return NewForwardOutputConfig("") })
}

// NewForwardOutputConfig creates a new forward output config with default values
func NewForwardOutputConfig(operatorID string) *ForwardOutputConfig {
	return &ForwardOutputConfig{
		OutputConfig:  helper.NewOutputConfig(operatorID, "forward_output"),
		BufferConfig:  buffer.NewConfig(),
		FlusherConfig: flusher.NewConfig(),
	}
}

// ForwardOutputConfig is the configuration of a forward output operator.
type ForwardOutputConfig struct {
	helper.OutputConfig `yaml:",inline"`
	BufferConfig        buffer.Config  `json:"buffer"  yaml:"buffer"`
	FlusherConfig       flusher.Config `json:"flusher" yaml:"flusher"`
	Address             string         `json:"address" yaml:"address"`
}

// Build will build an forward output operator.
func (c ForwardOutputConfig) Build(bc operator.BuildContext) ([]operator.Operator, error) {
	outputOperator, err := c.OutputConfig.Build(bc)
	if err != nil {
		return nil, err
	}

	buffer, err := c.BufferConfig.Build()
	if err != nil {
		return nil, err
	}

	if c.Address == "" {
		return nil, otelerrors.NewError("missing required parameter 'address'", "")
	}

	flusher := c.FlusherConfig.Build(bc.Logger.SugaredLogger)

	ctx, cancel := context.WithCancel(context.Background())

	forwardOutput := &ForwardOutput{
		OutputOperator: outputOperator,
		buffer:         buffer,
		flusher:        flusher,
		ctx:            ctx,
		cancel:         cancel,
		client:         &http.Client{},
		address:        c.Address,
	}

	return []operator.Operator{forwardOutput}, nil
}

// ForwardOutput is an operator that sends entries to another stanza instance
type ForwardOutput struct {
	helper.OutputOperator
	buffer  buffer.Buffer
	flusher *flusher.Flusher

	client  *http.Client
	address string

	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup
}

// Start signals to the ForwardOutput to begin flushing
func (f *ForwardOutput) Start(_ operator.Persister) error {
	f.wg.Add(1)
	go func() {
		defer f.wg.Done()
		f.feedFlusher(f.ctx)
	}()

	return nil
}

// Stop tells the ForwardOutput to stop gracefully
func (f *ForwardOutput) Stop() error {
	f.cancel()
	f.wg.Wait()
	f.flusher.Stop()
	// TODO handle buffer close entries
	entries, err := f.buffer.Close()
	if err != nil {
		f.Errorf("Failed to retreive entries")
		return err
	}

	err = f.handleCloseBuffer(err, entries)
	return err
}

func (f *ForwardOutput) handleCloseBuffer(e error, entries []*entry.Entry) error {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()

	req, err := f.createRequest(ctx, entries)
	if err != nil {
		f.Errorf("Failed to create request", zap.Error(err))
		return nil
	}

	res, err := f.client.Do(req)
	if err != nil {
		return otelerrors.Wrap(err, "send request")
	}

	if err := f.handleResponse(res); err != nil {
		return err
	}

	return nil
}

// Process adds an entry to the outputs buffer
func (f *ForwardOutput) Process(ctx context.Context, entry *entry.Entry) error {
	return f.buffer.Add(ctx, entry)
}

// ProcessMulti will send entries to elasticsearch.
func (f *ForwardOutput) createRequest(ctx context.Context, entries []*entry.Entry) (*http.Request, error) {
	var b bytes.Buffer
	enc := json.NewEncoder(&b)
	err := enc.Encode(entries)
	if err != nil {
		return nil, err
	}

	return http.NewRequestWithContext(ctx, "POST", f.address, &b)
}

func (f *ForwardOutput) feedFlusher(ctx context.Context) {
	for {
		entries, err := f.buffer.Read(ctx)
		switch {
		case errors.Is(err, context.Canceled):
			return
		case err != nil:
			f.Errorf("Failed to read chunk", zap.Error(err))
			continue
		}

		f.flusher.Do(ctx, func(flushCtx context.Context) error {
			req, err := f.createRequest(flushCtx, entries)
			if err != nil {
				f.Errorf("Failed to create request", zap.Error(err))
				return nil
			}

			res, err := f.client.Do(req)
			if err != nil {
				return otelerrors.Wrap(err, "send request")
			}

			if err := f.handleResponse(res); err != nil {
				return err
			}

			return nil
		})
	}
}

func (f *ForwardOutput) handleResponse(res *http.Response) error {
	if !(res.StatusCode >= 200 && res.StatusCode < 300) {
		body, err := ioutil.ReadAll(res.Body)
		if err != nil {
			return otelerrors.NewError("unexpected status code", "", "status", res.Status)
		} else {
			if err := res.Body.Close(); err != nil {
				f.Errorf(err.Error())
			}
			return otelerrors.NewError("unexpected status code", "", "status", res.Status, "body", string(body))
		}
	}
	return res.Body.Close()
}
