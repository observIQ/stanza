package forward

import (
	"context"
	"encoding/json"
	"net"
	"net/http"

	"github.com/observiq/stanza/v2/entry"
	"github.com/observiq/stanza/v2/errors"
	"github.com/observiq/stanza/v2/operator"
	"github.com/observiq/stanza/v2/operator/helper"
	"go.uber.org/zap"
)

func init() {
	operator.Register("forward_input", func() operator.Builder { return NewForwardInputConfig("") })
}

// NewForwardInputConfig creates a new stdin input config with default values
func NewForwardInputConfig(operatorID string) *ForwardInputConfig {
	return &ForwardInputConfig{
		InputConfig: helper.NewInputConfig(operatorID, "stdin"),
	}
}

// ForwardInputConfig is the configuration of a forward input operator
type ForwardInputConfig struct {
	helper.InputConfig `yaml:",inline"`
	ListenAddress      string     `json:"listen_address" yaml:"listen_address"`
	TLS                *TLSConfig `json:"tls"            yaml:"tls"`
}

// TLSConfig is a configuration struct for forward input TLS
type TLSConfig struct {
	CertFile string `json:"cert_file" yaml:"cert_file"`
	KeyFile  string `json:"key_file"  yaml:"key_file"`
}

// Build will build a forward input operator.
func (c *ForwardInputConfig) Build(context operator.BuildContext) ([]operator.Operator, error) {
	inputOperator, err := c.InputConfig.Build(context)
	if err != nil {
		return nil, err
	}

	forwardInput := &ForwardInput{
		InputOperator: inputOperator,
		tls:           c.TLS,
	}

	forwardInput.srv = &http.Server{
		Addr:    c.ListenAddress,
		Handler: forwardInput,
	}

	return []operator.Operator{forwardInput}, nil
}

// ForwardInput is an operator that reads input from stdin
type ForwardInput struct {
	helper.InputOperator

	srv *http.Server
	ln  net.Listener
	tls *TLSConfig
}

// Start will start generating log entries.
func (f *ForwardInput) Start() error {
	addr := f.srv.Addr
	if addr == "" {
		addr = ":http"
	}

	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return errors.Wrap(err, "start listener")
	}

	// Save the listener so we can use a dynamic port for tests
	f.ln = ln

	go func() {
		if f.tls != nil {
			err = f.srv.ServeTLS(ln, f.tls.CertFile, f.tls.KeyFile)
		} else {
			err = f.srv.Serve(ln)
		}
		if err != nil && err != http.ErrServerClosed {
			f.Errorw("Serve error", zap.Error(err))
		}
	}()

	return nil
}

// Stop will stop generating logs.
func (f *ForwardInput) Stop() error {
	return f.srv.Shutdown(context.Background())
}

func (f *ForwardInput) ServeHTTP(wr http.ResponseWriter, req *http.Request) {
	dec := json.NewDecoder(req.Body)

	var entries []*entry.Entry
	if err := dec.Decode(&entries); err != nil {
		wr.WriteHeader(http.StatusBadRequest)
		return
	}

	for _, entry := range entries {
		f.Write(req.Context(), entry)
	}
}
