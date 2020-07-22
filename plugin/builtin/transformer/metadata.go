package transformer

import (
	"context"

	"github.com/observiq/carbon/entry"
	"github.com/observiq/carbon/errors"
	"github.com/observiq/carbon/plugin"
	"github.com/observiq/carbon/plugin/helper"
)

func init() {
	plugin.Register("metadata", func() plugin.Builder { return NewMetadataOperatorConfig("") })
}

func NewMetadataOperatorConfig(pluginID string) *MetadataOperatorConfig {
	return &MetadataOperatorConfig{
		TransformerConfig: helper.NewTransformerConfig(pluginID, "metadata"),
	}
}

// MetadataOperatorConfig is the configuration of a metadata plugin
type MetadataOperatorConfig struct {
	helper.TransformerConfig `yaml:",inline"`

	Labels map[string]helper.ExprStringConfig `json:"labels" yaml:"labels"`
	Tags   []helper.ExprStringConfig          `json:"tags"   yaml:"tags"`
}

// Build will build a metadata plugin from the supplied configuration
func (c MetadataOperatorConfig) Build(context plugin.BuildContext) (plugin.Operator, error) {
	transformerOperator, err := c.TransformerConfig.Build(context)
	if err != nil {
		return nil, err
	}

	labeler, err := buildLabeler(c.Labels)
	if err != nil {
		return nil, errors.Wrap(err, "validate labels")
	}

	tagger, err := buildTagger(c.Tags)
	if err != nil {
		return nil, errors.Wrap(err, "validate labels")
	}

	restructureOperator := &MetadataOperator{
		TransformerOperator: transformerOperator,
		labeler:             labeler,
		tagger:              tagger,
	}

	return restructureOperator, nil
}

// MetadataOperator is a plugin that can add metadata to incoming entries
type MetadataOperator struct {
	helper.TransformerOperator
	labeler *labeler
	tagger  *tagger
}

// Process will process an incoming entry using the metadata transform.
func (p *MetadataOperator) Process(ctx context.Context, entry *entry.Entry) error {
	return p.ProcessWith(ctx, entry, p.Transform)
}

// Transform will transform an entry using the labeler and tagger.
func (p *MetadataOperator) Transform(entry *entry.Entry) (*entry.Entry, error) {
	err := p.labeler.Label(entry)
	if err != nil {
		return entry, err
	}

	err = p.tagger.Tag(entry)
	if err != nil {
		return entry, err
	}

	return entry, nil
}

type labeler struct {
	labels map[string]*helper.ExprString
}

// Label will add a label to an entry
func (l *labeler) Label(e *entry.Entry) error {
	env := helper.GetExprEnv(e)
	defer helper.PutExprEnv(env)

	for k, v := range l.labels {
		rendered, err := v.Render(env)
		if err != nil {
			return err
		}
		e.AddLabel(k, rendered)
	}

	return nil
}

func buildLabeler(config map[string]helper.ExprStringConfig) (*labeler, error) {
	labels := make(map[string]*helper.ExprString)

	for k, v := range config {
		exprString, err := v.Build()
		if err != nil {
			return nil, err
		}

		labels[k] = exprString
	}

	return &labeler{labels}, nil
}

type tagger struct {
	tags []*helper.ExprString
}

// Tag wil add a tag to an entry
func (t *tagger) Tag(e *entry.Entry) error {
	env := helper.GetExprEnv(e)
	defer helper.PutExprEnv(env)

	for _, v := range t.tags {
		rendered, err := v.Render(env)
		if err != nil {
			return err
		}
		e.Tags = append(e.Tags, rendered)
	}

	return nil
}

func buildTagger(config []helper.ExprStringConfig) (*tagger, error) {
	tags := make([]*helper.ExprString, 0, len(config))

	for _, v := range config {
		exprString, err := v.Build()
		if err != nil {
			return nil, err
		}

		tags = append(tags, exprString)
	}

	return &tagger{tags}, nil
}
