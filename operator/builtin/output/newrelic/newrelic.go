package newrelic

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"time"

	"github.com/observiq/stanza/entry"
	"github.com/observiq/stanza/errors"
	"github.com/observiq/stanza/operator"
	"github.com/observiq/stanza/operator/buffer"
	"github.com/observiq/stanza/operator/flusher"
	"github.com/observiq/stanza/operator/helper"
)

func init() {
	operator.RegisterOperator("newrelic_output", func() operator.Builder { return NewNewRelicOutputConfig("") })
}

// NewNewRelicOutputConfig creates a new elastic output config with default values
func NewNewRelicOutputConfig(operatorID string) *NewRelicOutputConfig {
	return &NewRelicOutputConfig{
		OutputConfig:  helper.NewOutputConfig(operatorID, "newrelic_output"),
		BufferConfig:  buffer.NewConfig(),
		FlusherConfig: flusher.NewConfig(),
		BaseURI:       "https://log-api.newrelic.com/log/v1",
		Timeout:       helper.NewDuration(10 * time.Second),
		MessageField:  entry.NewRecordField(),
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
func (c NewRelicOutputConfig) Build(context operator.BuildContext) (operator.Operator, error) {
	outputOperator, err := c.OutputConfig.Build(context)
	if err != nil {
		return nil, err
	}

	headers, err := c.getHeaders()
	if err != nil {
		return nil, err
	}

	buffer, err := c.BufferConfig.Build(context, c.ID())
	if err != nil {
		return nil, err
	}

	url, err := url.Parse(c.BaseURI)
	if err != nil {
		return nil, errors.Wrap(err, "'base_uri' is not a valid URL")
	}

	nro := &NewRelicOutput{
		OutputOperator: outputOperator,
		buffer:         buffer,
		client:         &http.Client{},
		headers:        headers,
		url:            url,
		timeout:        c.Timeout.Raw(),
		messageField:   c.MessageField,
	}

	nro.flusher = c.FlusherConfig.Build(buffer, nro.ProcessMulti, nro.SugaredLogger)

	return nro, nil
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

	client       *http.Client
	url          *url.URL
	headers      http.Header
	timeout      time.Duration
	messageField entry.Field
}

// Start tests the connection to New Relic and begins flushing entries
func (nro *NewRelicOutput) Start() error {
	// Test connection
	ctx, cancel := context.WithTimeout(context.Background(), nro.timeout)
	defer cancel()
	if err := nro.ProcessMulti(ctx, nil); err != nil {
		return errors.Wrap(err, "test connection")
	}

	nro.flusher.Start()
	return nil
}

// Stop tells the NewRelicOutput to stop gracefully
func (nro *NewRelicOutput) Stop() error {
	nro.flusher.Stop()
	return nro.buffer.Close()
}

// Process adds an entry to the output's buffer
func (nro *NewRelicOutput) Process(ctx context.Context, entry *entry.Entry) error {
	return nro.buffer.Add(ctx, entry)
}

// ProcessMulti will send a chunk of entries to New Relic
func (nro *NewRelicOutput) ProcessMulti(ctx context.Context, entries []*entry.Entry) error {
	lp := LogPayloadFromEntries(entries, nro.messageField)

	ctx, cancel := context.WithTimeout(ctx, nro.timeout)
	defer cancel()
	req, err := nro.newRequest(ctx, lp)
	if err != nil {
		return errors.Wrap(err, "create request")
	}

	res, err := nro.client.Do(req)
	if err != nil {
		return errors.Wrap(err, "execute request")
	}
	nro.handleResponse(res)
	return nil
}

// newRequest creates a new http.Request with the given context and payload
func (nro *NewRelicOutput) newRequest(ctx context.Context, payload LogPayload) (*http.Request, error) {
	var buf bytes.Buffer
	wr := gzip.NewWriter(&buf)
	enc := json.NewEncoder(wr)
	if err := enc.Encode(payload); err != nil {
		return nil, errors.Wrap(err, "encode payload")
	}
	if err := wr.Close(); err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", nro.url.String(), &buf)
	if err != nil {
		return nil, err
	}
	req.Header = nro.headers

	return req, nil
}

func (nro *NewRelicOutput) handleResponse(res *http.Response) {
	if !(res.StatusCode >= 200 && res.StatusCode < 300) {
		body, err := ioutil.ReadAll(res.Body)
		if err != nil {
			nro.Errorw("Request returned a non-zero status code", "status", res.Status)
		} else {
			nro.Errorw("Request returned a non-zero status code", "status", res.Status, "body", body)
		}
	}
	res.Body.Close()
}
