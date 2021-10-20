package replace

import (
	"context"
	"fmt"
	"strings"

	"github.com/observiq/stanza/entry"
	"github.com/observiq/stanza/errors"
	"github.com/observiq/stanza/operator"
	"github.com/observiq/stanza/operator/helper"
)

func init() {
	operator.Register("string_replace", func() operator.Builder { return NewStringReplaceOperatorConfig("") })
}

// NewStringReplaceOperatorConfig creates a new replace operator config with default values
func NewStringReplaceOperatorConfig(operatorID string) *StringReplaceOperatorConfig {
	return &ReplaceParserConfig{
		TransformerConfig: helper.NewTransformerConfig(operatorID, "string_replace"),
	}
}

// StringReplaceOperatorConfig is the configuration of a regex parser operator.
type StringReplaceOperatorConfig struct {
	helper.TransformerConfig `mapstructure:",squash" yaml:",inline"`
	String string `mapstructure:"string" json:"string" yaml:"string"`
	Replacement string `mapstructure:"value,omitempty" json:"value,omitempty" yaml:"value,omitempty"
}

// Build will build a regex parser operator.
func (c StringReplaceOperatorConfig) Build(context operator.BuildContext) ([]operator.Operator, error) {
	transformerOperator, err := c.TransformerConfig.Build(context)
	if err != nil {
		return nil, err
	}

	if c.String == "" {
		return nil, fmt.Errorf("missing required field 'String'")
	}

	if c.Replacement == "" {
		return nil, fmt.Errorf("missing required field 'Replacement'")
	}

	string_replaceOperator := &string_replaceOperator{
                TransformerOperator: transformerOperator
                String:              c.String
                Replacement:	     c.Replacement
        }

	return []operator.Operator{string_replaceOperator}, nil
}

// StringReplaceOperator is an operator that performs string replacement in an entry.
type StringReplaceOperator struct {
	helper.TransformerOperator
	String	    string
	Replacement string
}

// Process will parse an entry for string replacement.
func (sr *StringReplaceOperator) Process(ctx context.Context, entry *entry.Entry) error {
	return sr.ProcessWith(ctx, entry, sr.Transform)
}

// Transform will parse a value using the supplied strings and perform
// the appropriate string replacements
func (sr *StringReplaceOperator) Transform(Value string) (error) {
	return strings.ReplaceAll(Value, sr.String, sr.Replacement)
}
