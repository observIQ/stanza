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

	"github.com/observiq/stanza/v2/version"
	otelerrors "github.com/open-telemetry/opentelemetry-log-collection/errors"
)

// Client is an interface for sending a log payload to new relic
type client interface {
	SendPayload(context.Context, LogPayload) error
	TestConnection(context.Context) error
}

// client is the standard implementation of the Client interface
type nroClient struct {
	endpoint   *url.URL
	headers    http.Header
	httpClient *http.Client
}

// NewClient creates a standard client for sending logs to new relic
func newClient(endpoint *url.URL, headers http.Header) client {
	return &nroClient{
		endpoint:   endpoint,
		headers:    headers,
		httpClient: &http.Client{},
	}
}

// SendPayload creates an http request from a log payload and sends it to new relic
func (c *nroClient) SendPayload(ctx context.Context, payload LogPayload) error {
	req, err := c.createRequest(ctx, payload)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	res, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}

	return c.checkResponse(res)
}

// TestConnection tests the connection to the new relic api
func (c *nroClient) TestConnection(ctx context.Context) error {
	logs := make([]*LogMessage, 0, 0)
	payload := LogPayload{{
		Common: LogPayloadCommon{
			Attributes: map[string]interface{}{
				"plugin": map[string]interface{}{
					"type":    "stanza",
					"version": version.GetVersion(),
				},
			},
		},
		Logs: logs,
	}}

	err := c.SendPayload(ctx, payload)
	if err != nil {
		return fmt.Errorf("failed to send empty payload: %w", err)
	}

	return nil
}

// createRequest creates a new http.Request with the given context and log payload
func (c *nroClient) createRequest(ctx context.Context, payload LogPayload) (*http.Request, error) {
	var buf bytes.Buffer
	wr := gzip.NewWriter(&buf)
	enc := json.NewEncoder(wr)
	if err := enc.Encode(payload); err != nil {
		return nil, otelerrors.Wrap(err, "encode payload")
	}
	if err := wr.Close(); err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", c.endpoint.String(), &buf)
	if err != nil {
		return nil, err
	}
	req.Header = c.headers

	return req, nil
}

// checkResponse checks a response from the new relic api
func (c *nroClient) checkResponse(res *http.Response) error {
	defer func() {
		_ = res.Body.Close()
	}()

	if !(res.StatusCode >= 200 && res.StatusCode < 300) {
		body, err := ioutil.ReadAll(res.Body)
		if err != nil {
			return otelerrors.NewError("unexpected status code", "", "status", res.Status)
		}
		return otelerrors.NewError("unexpected status code", "", "status", res.Status, "body", string(body))
	}
	return nil
}
