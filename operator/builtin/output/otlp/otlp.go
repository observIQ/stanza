package otlp

import (
	"bytes"
	"context"
	"io/ioutil"
	"net/http"
	"net/url"

	"github.com/observiq/stanza/entry"
	"github.com/observiq/stanza/errors"
	"github.com/observiq/stanza/operator"
	"github.com/observiq/stanza/operator/buffer"
	"github.com/observiq/stanza/operator/flusher"
	"github.com/observiq/stanza/operator/helper"
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
func (c OTLPOutputConfig) Build(context operator.BuildContext) ([]operator.Operator, error) {

	outputOperator, err := c.OutputConfig.Build(context)
	if err != nil {
		return nil, err
	}

	buffer, err := c.BufferConfig.Build(context, c.ID())
	if err != nil {
		return nil, err
	}

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

	otlp := &OTLPOutput{
		OutputOperator: outputOperator,
		buffer:         buffer,
		client:         client,
		url:            url,
	}

	otlp.flusher = c.FlusherConfig.Build(buffer, otlp.ProcessMulti, otlp.SugaredLogger)

	return []operator.Operator{otlp}, nil
}

// OTLPOutput is an operator that sends entries to the OTLP recevier
type OTLPOutput struct {
	helper.OutputOperator
	buffer  buffer.Buffer
	flusher *flusher.Flusher
	client  *http.Client
	url     *url.URL
}

// Start flushing entries
func (o *OTLPOutput) Start() error {
	o.flusher.Start()
	return nil
}

// Stop tells the OTLPOutput to stop gracefully
func (o *OTLPOutput) Stop() error {
	o.flusher.Stop()
	return o.buffer.Close()
}

// Process adds an entry to the output's buffer
func (o *OTLPOutput) Process(ctx context.Context, entry *entry.Entry) error {
	return o.buffer.Add(ctx, entry)
}

// ProcessMulti will send a chunk of entries
func (o *OTLPOutput) ProcessMulti(ctx context.Context, entries []*entry.Entry) error {

	logs := Convert(entries)
	protoBytes, err := logs.ToOtlpProtoBytes()
	if err != nil {
		return errors.Wrap(err, "convert logs to proto bytes")
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, o.url.String(), bytes.NewReader(protoBytes))
	if err != nil {
		return errors.Wrap(err, "create request")
	}
	req.Header.Set("Content-Type", "application/x-protobuf")

	res, err := o.client.Do(req)
	if err != nil {
		return errors.Wrap(err, "send request")
	}

	o.handleResponse(res)
	return err
}

func (o *OTLPOutput) handleResponse(res *http.Response) {
	if !(res.StatusCode >= 200 && res.StatusCode < 300) {
		body, err := ioutil.ReadAll(res.Body)
		if err != nil {
			o.Errorw("Request returned a non-zero status code", "status", res.Status)
		} else {
			o.Errorw("Request returned a non-zero status code", "status", res.Status, "body", body)
		}
	}
	res.Body.Close()
}
