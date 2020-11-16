package otlp

import (
	"bytes"
	"context"
	"io/ioutil"
	"net/http"
	"net/url"
  "sync"

	"github.com/observiq/stanza/entry"
	"github.com/observiq/stanza/errors"
	"github.com/observiq/stanza/operator"
	"github.com/observiq/stanza/operator/buffer"
	"github.com/observiq/stanza/operator/flusher"
	"github.com/observiq/stanza/operator/helper"
  "go.uber.org/zap"
)

func init() {
	operator.Register("otlp_output", func() operator.Builder { return NewOTLPOutputConfig("") })
}

// NewOTLPOutputConfig creates a new elastic output config with default values
func NewOTLPOutputConfig(operatorID string) *OTLPOutputConfig {
	return &OTLPOutputConfig{
		OutputConfig:     helper.NewOutputConfig(operatorID, "otlp_output"),
		BufferConfig:     buffer.NewConfig(),
		FlusherConfig:    flusher.NewConfig(),
		HTTPClientConfig: NewHTTPClientConfig(),
	}
}

// OTLPOutputConfig is the configuration of a OTLPOutput operator
type OTLPOutputConfig struct {
	helper.OutputConfig `yaml:",inline"`
	BufferConfig        buffer.Config  `json:"buffer" yaml:"buffer"`
	FlusherConfig       flusher.Config `json:"flusher" yaml:"flusher"`
	HTTPClientConfig    `yaml:",inline"`
}

// Build will build a new OTLPOutput
func (c OTLPOutputConfig) Build(bc operator.BuildContext) ([]operator.Operator, error) {

	outputOperator, err := c.OutputConfig.Build(bc)
	if err != nil {
		return nil, err
	}

	buffer, err := c.BufferConfig.Build(bc, c.ID())
	if err != nil {
		return nil, err
	}

  flusher := c.FlusherConfig.Build(bc.Logger.SugaredLogger)

	if err := c.cleanEndpoint(); err != nil {
		return nil, err
	}

	client, err := c.HTTPClientConfig.ToClient()
	if err != nil {
		return nil, errors.Wrap(err, "create client")
	}

	url, err := url.Parse(c.HTTPClientConfig.Endpoint)
	if err != nil {
		return nil, errors.Wrap(err, "'endpoint' is not a valid URL")
	}

  ctx, cancel := context.WithCancel(context.Background())

	otlp := &OTLPOutput{
		OutputOperator: outputOperator,
		buffer:         buffer,
    flusher: flusher,
		client:         client,
		url:            url,
    ctx: ctx,
    cancel: cancel,
	}


	return []operator.Operator{otlp}, nil
}

// OTLPOutput is an operator that sends entries to the OTLP recevier
type OTLPOutput struct {
	helper.OutputOperator
	buffer  buffer.Buffer
	flusher *flusher.Flusher
	client  *http.Client
	url     *url.URL

  ctx context.Context
  cancel context.CancelFunc
  wg sync.WaitGroup
}

// Start flushing entries
func (o *OTLPOutput) Start() error {
  o.wg.Add(1)
  go func() {
    defer o.wg.Done()
    o.feedFlusher(o.ctx)
  }()

	return nil
}

// Stop tells the OTLPOutput to stop gracefully
func (o *OTLPOutput) Stop() error {
  o.cancel()
  o.wg.Wait()
	o.flusher.Stop()
	return o.buffer.Close()
}

func (o *OTLPOutput) feedFlusher(ctx context.Context) {
	for {
    // Get the next chunk of entries
		entries, flushFunc, err := o.buffer.ReadChunk(ctx, 1000)
		if err != nil && err == context.Canceled {
			return
		} else if err != nil {
			o.Errorf("Failed to read chunk", zap.Error(err))
			continue
		}

		req, err := o.createRequest(ctx, entries)
		o.flusher.Do(func(ctx context.Context) error {
      res, err := o.client.Do(req)
      if err != nil {
        return errors.Wrap(err, "send request")
      }

      if err := o.handleResponse(res); err != nil {
        return err
      }

			if err = flushFunc(); err != nil {
				o.Errorw("Failed to mark entries as flushed", zap.Error(err))
			}
			return nil
		})
	}
}

// Process adds an entry to the output's buffer
func (o *OTLPOutput) Process(ctx context.Context, entry *entry.Entry) error {
	return o.buffer.Add(ctx, entry)
}

// ProcessMulti will send a chunk of entries
func (o *OTLPOutput) createRequest(ctx context.Context, entries []*entry.Entry) (*http.Request, error) {
	logs := Convert(entries)
	protoBytes, err := logs.ToOtlpProtoBytes()
	if err != nil {
		return nil, errors.Wrap(err, "convert logs to proto bytes")
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, o.url.String(), bytes.NewReader(protoBytes))
	if err != nil {
		return nil, errors.Wrap(err, "create request")
	}
	req.Header.Set("Content-Type", "application/x-protobuf")
  return req, nil
}

func (o *OTLPOutput) handleResponse(res *http.Response) error {
	if !(res.StatusCode >= 200 && res.StatusCode < 300) {
		body, err := ioutil.ReadAll(res.Body)
		if err != nil {
      return errors.NewError("non-success status code", "", "status", string(res.StatusCode))
		} else {
      res.Body.Close()
      return errors.NewError("non-success status code", "", "status", string(res.StatusCode), "body", string(body))
		}
	}
  res.Body.Close()
  return nil
}
