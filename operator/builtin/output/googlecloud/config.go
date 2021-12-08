package googlecloud

import (
	"context"
	"errors"
	"fmt"
	"io/ioutil"
	"time"

	"github.com/observiq/stanza/entry"
	"github.com/observiq/stanza/operator"
	"github.com/observiq/stanza/operator/buffer"
	"github.com/observiq/stanza/operator/flusher"
	"github.com/observiq/stanza/operator/helper"
	"github.com/observiq/stanza/version"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/option"
	"google.golang.org/grpc"
	"google.golang.org/grpc/encoding/gzip"
)

const (
	operatorType          = "google_cloud_output"
	credentialScope       = "https://www.googleapis.com/auth/logging.write"
	defaultTimeout        = 30 * time.Second
	defaultUseCompression = true
	defaultMaxEntrySize   = 200000
	defaultMaxRequestSize = 5000000
)

func init() {
	operator.Register(operatorType, func() operator.Builder { return NewGoogleCloudOutputConfig("") })
}

// NewGoogleCloudOutputConfig creates a new google cloud output config with default
func NewGoogleCloudOutputConfig(operatorID string) *GoogleCloudOutputConfig {
	return &GoogleCloudOutputConfig{
		OutputConfig:   helper.NewOutputConfig(operatorID, operatorType),
		BufferConfig:   buffer.NewConfig(),
		FlusherConfig:  flusher.NewConfig(),
		Timeout:        helper.Duration{Duration: defaultTimeout},
		UseCompression: defaultUseCompression,
		MaxEntrySize:   defaultMaxEntrySize,
		MaxRequestSize: defaultMaxRequestSize,
	}
}

// GoogleCloudOutputConfig is the configuration of a google cloud output operator.
type GoogleCloudOutputConfig struct {
	helper.OutputConfig `yaml:",inline"`
	BufferConfig        buffer.Config   `json:"buffer,omitempty" yaml:"buffer,omitempty"`
	FlusherConfig       flusher.Config  `json:"flusher,omitempty" yaml:"flusher,omitempty"`
	Credentials         string          `json:"credentials,omitempty"      yaml:"credentials,omitempty"`
	CredentialsFile     string          `json:"credentials_file,omitempty" yaml:"credentials_file,omitempty"`
	ProjectID           string          `json:"project_id"                 yaml:"project_id"`
	LogNameField        *entry.Field    `json:"log_name_field,omitempty"   yaml:"log_name_field,omitempty"`
	LocationField       *entry.Field    `json:"location_field,omitempty"   yaml:"location_field,omitempty"`
	TraceField          *entry.Field    `json:"trace_field,omitempty"      yaml:"trace_field,omitempty"`
	SpanIDField         *entry.Field    `json:"span_id_field,omitempty"    yaml:"span_id_field,omitempty"`
	Timeout             helper.Duration `json:"timeout,omitempty"          yaml:"timeout,omitempty"`
	UseCompression      bool            `json:"use_compression,omitempty"  yaml:"use_compression,omitempty"`
	MaxEntrySize        helper.ByteSize `json:"max_entry_size"             yaml:"max_entry_size"`
	MaxRequestSize      helper.ByteSize `json:"max_request_size"           yaml:"max_request_size"`
}

// Build will build a google cloud output operator.
func (c GoogleCloudOutputConfig) Build(bc operator.BuildContext) ([]operator.Operator, error) {
	outputOperator, err := c.OutputConfig.Build(bc)
	if err != nil {
		return nil, fmt.Errorf("failed to build output operator: %w", err)
	}

	newBuffer, err := c.BufferConfig.Build(bc, c.ID())
	if err != nil {
		return nil, fmt.Errorf("failed to build buffer: %w", err)
	}

	credentials, err := c.getCredentials()
	if err != nil {
		return nil, fmt.Errorf("failed to get credentials: %w", err)
	}

	var projectID string
	switch {
	case c.ProjectID != "":
		projectID = c.ProjectID
	case credentials.ProjectID != "":
		projectID = credentials.ProjectID
	default:
		return nil, errors.New("failed to get project id from config or credentials")
	}

	newFlusher := c.FlusherConfig.Build(bc.Logger.SugaredLogger)
	clientOptions := c.createClientOptions(credentials, c.UseCompression)
	ctx, cancel := context.WithCancel(context.Background())

	entryBuilder := &GoogleEntryBuilder{
		MaxEntrySize:  int(c.MaxEntrySize),
		ProjectID:     projectID,
		LogNameField:  c.LogNameField,
		LocationField: c.LocationField,
		TraceField:    c.TraceField,
		SpanIDField:   c.SpanIDField,
	}

	requestBuilder := &GoogleRequestBuilder{
		MaxRequestSize: int(c.MaxRequestSize),
		ProjectID:      projectID,
		EntryBuilder:   entryBuilder,
		SugaredLogger:  outputOperator.SugaredLogger.Named("RequestBuilder"),
	}

	googleCloudOutput := &GoogleCloudOutput{
		OutputOperator: outputOperator,
		buffer:         newBuffer,
		flusher:        newFlusher,
		timeout:        c.Timeout.Raw(),
		requestBuilder: requestBuilder,
		clientOptions:  clientOptions,
		buildClient:    newClient,
		ctx:            ctx,
		cancel:         cancel,
	}

	return []operator.Operator{googleCloudOutput}, nil
}

// getCredentials parses the google cloud logging credentials specified in the config.
// If credentials and credentials_file are both specified, credentials will take precedence.
// If neither field is specified, an attempt is made to locate the default credentials.
func (c GoogleCloudOutputConfig) getCredentials() (*google.Credentials, error) {
	switch {
	case c.Credentials != "":
		credentials, err := google.CredentialsFromJSON(context.Background(), []byte(c.Credentials), credentialScope)
		if err != nil {
			return nil, fmt.Errorf("failed to parse credentials field: %w", err)
		}

		return credentials, nil
	case c.CredentialsFile != "":
		bytes, err := ioutil.ReadFile(c.CredentialsFile)
		if err != nil {
			return nil, fmt.Errorf("failed to read credentials file: %w", err)
		}

		credentials, err := google.CredentialsFromJSON(context.Background(), bytes, credentialScope)
		if err != nil {
			return nil, fmt.Errorf("failed to parse credentials in credentials file: %w", err)
		}

		return credentials, nil
	default:
		credentials, err := google.FindDefaultCredentials(context.Background(), credentialScope)
		if err != nil {
			return nil, fmt.Errorf("failed to find default credentials: %w", err)
		}

		return credentials, nil
	}
}

// createClientOptions creates client options from the supplied values
func (c GoogleCloudOutputConfig) createClientOptions(credentials *google.Credentials, useCompression bool) []option.ClientOption {
	options := make([]option.ClientOption, 0, 2)
	options = append(options, option.WithCredentials(credentials))
	options = append(options, option.WithUserAgent("StanzaLogAgent/"+version.GetVersion()))
	if useCompression {
		compressOption := option.WithGRPCDialOption(grpc.WithDefaultCallOptions(grpc.UseCompressor(gzip.Name)))
		options = append(options, compressOption)
	}

	return options
}
