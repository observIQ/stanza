package elastic

import (
	"bytes"
	"context"
  "net/http"
	"encoding/json"
	"sync"
  "io/ioutil"

	"github.com/observiq/stanza/entry"
	"github.com/observiq/stanza/errors"
	"github.com/observiq/stanza/operator"
	"github.com/observiq/stanza/operator/buffer"
	"github.com/observiq/stanza/operator/flusher"
	"github.com/observiq/stanza/operator/helper"
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

	buffer, err := c.BufferConfig.Build(bc, c.ID())
	if err != nil {
		return nil, err
	}

	flusher := c.FlusherConfig.Build(bc.Logger.SugaredLogger)

	ctx, cancel := context.WithCancel(context.Background())

	forwardOutput := &ForwardOutput{
		OutputOperator: outputOperator,
		buffer:         buffer,
		flusher:        flusher,
		ctx:            ctx,
		cancel:         cancel,
    client: &http.Client{},
    address: c.Address,
	}

	return []operator.Operator{forwardOutput}, nil
}

// ForwardOutput is an operator that sends entries to another stanza instance
type ForwardOutput struct {
	helper.OutputOperator
	buffer  buffer.Buffer
	flusher *flusher.Flusher

  client *http.Client
  address string

	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup
}

// Start signals to the ForwardOutput to begin flushing
func (f *ForwardOutput) Start() error {
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
	return f.buffer.Close()
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
		entries, clearer, err := f.buffer.ReadChunk(ctx)
		if err != nil && err == context.Canceled {
			return
		} else if err != nil {
			f.Errorf("Failed to read chunk", zap.Error(err))
			continue
		}

		req, err := f.createRequest(ctx, entries)
    if err != nil {
      f.Errorf("Failed to create request", zap.Error(err))
      continue
    }

		f.flusher.Do(func(ctx context.Context) error {
			res, err := f.client.Do(req)
			if err != nil {
				return errors.Wrap(err, "send request")
			}

      if err := f.handleResponse(res); err != nil {
        return err
      }

			if err = clearer.MarkAllAsFlushed(); err != nil {
				f.Errorw("Failed to mark entries as flushed", zap.Error(err))
			}
			return nil
		})
	}
}

func (f *ForwardOutput) handleResponse(res *http.Response) error {
	if !(res.StatusCode >= 200 && res.StatusCode < 300) {
		body, err := ioutil.ReadAll(res.Body)
		if err != nil {
			return errors.NewError("unexpected status code", "", "status", res.Status)
		} else {
			res.Body.Close()
			return errors.NewError("unexpected status code", "", "status", res.Status, "body", string(body))
		}
	}
	res.Body.Close()
	return nil
}
