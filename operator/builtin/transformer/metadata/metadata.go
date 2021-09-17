package metadata

import (
	"context"

	"github.com/observiq/stanza/entry"
	"github.com/observiq/stanza/errors"
	"github.com/observiq/stanza/operator"
	"github.com/observiq/stanza/operator/helper"
)

func init() {
	operator.Register("metadata", func() operator.Builder { return NewMetadataOperatorConfig("") })
}

// NewMetadataOperatorConfig creates a new metadata config with default values
func NewMetadataOperatorConfig(operatorID string) *MetadataOperatorConfig {
	return &MetadataOperatorConfig{
		TransformerConfig: helper.NewTransformerConfig(operatorID, "metadata"),
		AttributerConfig:  helper.NewAttributerConfig(),
		IdentifierConfig:  helper.NewIdentifierConfig(),
	}
}

// MetadataOperatorConfig is the configuration of a metadata operator
type MetadataOperatorConfig struct {
	helper.TransformerConfig `yaml:",inline"`
	helper.AttributerConfig  `yaml:",inline"`
	helper.IdentifierConfig  `yaml:",inline"`
}

// Build will build a metadata operator from the supplied configuration
func (c MetadataOperatorConfig) Build(context operator.BuildContext) ([]operator.Operator, error) {
	transformerOperator, err := c.TransformerConfig.Build(context)
	if err != nil {
		return nil, errors.Wrap(err, "failed to build transformer")
	}

	attributer, err := c.AttributerConfig.Build()
	if err != nil {
		return nil, errors.Wrap(err, "failed to build attributer")
	}

	identifier, err := c.IdentifierConfig.Build()
	if err != nil {
		return nil, errors.Wrap(err, "failed to build identifier")
	}

	metadataOperator := &MetadataOperator{
		TransformerOperator: transformerOperator,
		Attributer:          attributer,
		Identifier:          identifier,
	}

	return []operator.Operator{metadataOperator}, nil
}

// MetadataOperator is an operator that can add metadata to incoming entries
type MetadataOperator struct {
	helper.TransformerOperator
	helper.Attributer
	helper.Identifier
}

// Process will process an incoming entry using the metadata transform.
func (p *MetadataOperator) Process(ctx context.Context, entry *entry.Entry) error {
	return p.ProcessWith(ctx, entry, p.Transform)
}

// Transform will transform an entry using the attributer and tagger.
func (p *MetadataOperator) Transform(entry *entry.Entry) error {
	if err := p.Attribute(entry); err != nil {
		return errors.Wrap(err, "failed to add attributes to entry")
	}

	if err := p.Identify(entry); err != nil {
		return errors.Wrap(err, "failed to add resource keys to entry")
	}

	return nil
}
