package otlp

import (
	"context"
	"fmt"

	"github.com/observiq/stanza/entry"
	"github.com/observiq/stanza/operator"
	"github.com/observiq/stanza/operator/helper"
	"go.opentelemetry.io/collector/model/otlpgrpc"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
)

func init() {
	operator.Register("otlp", func() operator.Builder { return NewOTLPConfig("") })
}

const authorization = "authorization"

// NewOTLPConfig creates a new otlp output config with default values
func NewOTLPConfig(operatorID string) *OtlpConfig {
	return &OtlpConfig{
		OutputConfig: helper.NewOutputConfig(operatorID, "otlp_output"),
	}
}

// Build will build a otlp output operator.
func (c OtlpConfig) Build(context operator.BuildContext) ([]operator.Operator, error) {
	outputOperator, err := c.OutputConfig.Build(context)
	if err != nil {
		return nil, err
	}

	if c.Endpoint == "" {
		return nil, fmt.Errorf("must provide an endpoint")
	}

	otlpOutput := &OtlpOutput{
		OutputOperator: outputOperator,
		config:         c,
	}

	return []operator.Operator{otlpOutput}, nil
}

// OtlpOutput is an operator that writes logs to a service.
type OtlpOutput struct {
	helper.OutputOperator
	config     OtlpConfig
	logsClient otlpgrpc.LogsClient
	clientConn *grpc.ClientConn
}

// Start will open the connection.
func (o *OtlpOutput) Start() error {
	var opts []grpc.DialOption
	var err error
	ctx := context.Background()

	if !o.config.RetrySettings.Enabled {
		opts = append(opts, grpc.WithDisableRetry())
	}

	opts = append(opts, grpc.WithTransportCredentials(insecure.NewCredentials()))

	if o.config.Timeout > 0 {
		ctx, _ = context.WithTimeout(ctx, o.config.Timeout)
	}

	o.clientConn, err = grpc.DialContext(ctx, o.config.Endpoint, opts...)
	if err != nil {
		return err
	}
	o.logsClient = otlpgrpc.NewLogsClient(o.clientConn)
	return nil
}

// Stop will close the connection.
func (o *OtlpOutput) Stop() error {
	return o.clientConn.Close()
}

// Process will write an entry to the endpoint.
func (o *OtlpOutput) Process(ctx context.Context, entry *entry.Entry) error {
	md := metadata.New(map[string]string{authorization: o.config.Authorization})
	ctx = metadata.NewOutgoingContext(ctx, md)
	
	logRequest := otlpgrpc.NewLogsRequest()
	logRequest.SetLogs(convert(entry))
	_, err := o.logsClient.Export(ctx, logRequest)

	return err
}
