package transformer

import (
	"context"

	"github.com/observiq/carbon/entry"
	"github.com/observiq/carbon/errors"
	"github.com/observiq/carbon/operator"
	"github.com/observiq/carbon/operator/helper"
)

func init() {
	operator.Register("metadata", func() operator.Builder { return NewMetadataOperatorConfig("") })
}

// NewMetadataOperatorConfig creates a new metadata config with default values
func NewMetadataOperatorConfig(operatorID string) *MetadataOperatorConfig {
	return &MetadataOperatorConfig{
		TransformerConfig: helper.NewTransformerConfig(operatorID, "metadata"),
		LabelerConfig:     helper.NewLabelerConfig(),
	}
}

// MetadataOperatorConfig is the configuration of a metadata operator
type MetadataOperatorConfig struct {
	helper.TransformerConfig `yaml:",inline"`
	helper.LabelerConfig     `yaml:",inline"`
}

// Build will build a metadata operator from the supplied configuration
func (c MetadataOperatorConfig) Build(context operator.BuildContext) (operator.Operator, error) {
	transformerOperator, err := c.TransformerConfig.Build(context)
	if err != nil {
		return nil, errors.Wrap(err, "failed to build transformer")
	}

	labeler, err := c.LabelerConfig.Build()
	if err != nil {
		return nil, errors.Wrap(err, "failed to build labeler")
	}

	metadataOperator := &MetadataOperator{
		TransformerOperator: transformerOperator,
		Labeler:             labeler,
	}

	return metadataOperator, nil
}

// MetadataOperator is an operator that can add metadata to incoming entries
type MetadataOperator struct {
	helper.TransformerOperator
	helper.Labeler
}

// Process will process an incoming entry using the metadata transform.
func (p *MetadataOperator) Process(ctx context.Context, entry *entry.Entry) error {
	return p.ProcessWith(ctx, entry, p.Transform)
}

// Transform will transform an entry using the labeler and tagger.
func (p *MetadataOperator) Transform(entry *entry.Entry) (*entry.Entry, error) {
	err := p.Label(entry)
	if err != nil {
		return entry, err
	}

	return entry, nil
}
