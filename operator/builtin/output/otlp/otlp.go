package otlp

import (
	"context"

	"github.com/observiq/stanza/entry"
	"github.com/observiq/stanza/errors"
	"github.com/observiq/stanza/operator"
	"github.com/observiq/stanza/operator/buffer"
	"github.com/observiq/stanza/operator/flusher"
	"github.com/observiq/stanza/operator/helper"
	"google.golang.org/grpc"
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
		GRPCClientConfig: NewGRPCClientConfig(),
	}
}

// OTLPOutputConfig is the configuration of a OTLPOutput operator
type OTLPOutputConfig struct {
	helper.OutputConfig `yaml:",inline"`
	BufferConfig        buffer.Config  `json:"buffer" yaml:"buffer"`
	FlusherConfig       flusher.Config `json:"flusher" yaml:"flusher"`
	GRPCClientConfig    `yaml:",inline"`
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

	dialOpts, err := c.ToDialOptions()
	if err != nil {
		return nil, err
	}

	clientConn, err := grpc.Dial(c.Endpoint, dialOpts...)
	if err != nil {
		return nil, err
	}

	otlp := &OTLPOutput{
		OutputOperator:   outputOperator,
		buffer:           buffer,
		grpcClientConfig: c.GRPCClientConfig,
	}

	otlp.flusher = c.FlusherConfig.Build(buffer, otlp.ProcessMulti, otlp.SugaredLogger)

	return []operator.Operator{otlp}, nil
}

// OTLPOutput is an operator that sends entries to the New Relic Logs platform
type OTLPOutput struct {
	helper.OutputOperator
	buffer           buffer.Buffer
	flusher          *flusher.Flusher
	grpcClientConfig GRPCClientConfig
	client           *grpc.ClientConn
}

// Start tests the connection to New Relic and begins flushing entries
func (o *OTLPOutput) Start() error {
	// Test connection
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	if err := o.ProcessMulti(ctx, nil); err != nil {
		return errors.Wrap(err, "test connection")
	}

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

// ProcessMulti will send a chunk of entries to New Relic
func (o *OTLPOutput) ProcessMulti(ctx context.Context, entries []*entry.Entry) error {

	// TODO convert from entries to payload

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	req, err := o.newRequest(ctx, lp)
	if err != nil {
		return errors.Wrap(err, "create request")
	}

	callOpts := grpc.WaitForReady(o.grpcClientConfig.WaitForReady)
	return o.client.Invoke(ctx, "/opentelemetry.proto.collector.logs.v1.LogsService/Export", request, response, callOpts)
	// TODO possibly interpret error
}
