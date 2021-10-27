package regex_replace

import (
	"context"
	"fmt"
	"regexp"

	"github.com/observiq/stanza/entry"
	"github.com/observiq/stanza/errors"
	"github.com/observiq/stanza/operator"
	"github.com/observiq/stanza/operator/helper"
)

func init() {
	operator.Register("replace", func() operator.Builder { return NewReplaceOperatorConfig("") })
}

// NewReplaceOperatorConfig creates a new replace operator config with default values
func NewReplaceOperatorConfig(operatorID string) *ReplaceOperatorConfig {
	return &ReplaceParserConfig{
		TransformerConfig: helper.NewTransformerConfig(operatorID, "replace"),
	}
}

// ReplaceOperatorConfig is the configuration of a regex parser operator.
type ReplaceOperatorConfig struct {
	helper.TransformerConfig `mapstructure:",squash" yaml:",inline"`
	Regex string `mapstructure:"regex" json:"regex" yaml:"regex"`
	Value interface{} `mapstructure:"value,omitempty" json:"value,omitempty" yaml:"value,omitempty"`
}

// Build will build a regex parser operator.
func (c ReplaceOperatorConfig) Build(context operator.BuildContext) ([]operator.Operator, error) {
	transformerOperator, err := c.TransformerConfig.Build(context)
	if err != nil {
		return nil, err
	}

	if c.Regex == "" {
		return nil, fmt.Errorf("missing required field 'regex'")
	}

	r, err := regexp.Compile(c.Regex)
	if err != nil {
		return nil, fmt.Errorf("compiling regex: %s", err)
	}

	namedCaptureGroups := 0
	for _, groupName := range r.SubexpNames() {
		if groupName != "" {
			namedCaptureGroups++
		}
	}
	if namedCaptureGroups == 0 {
		return nil, errors.NewError(
			"no named capture groups in regex pattern",
			"use named capture groups like '^(?P<my_key>.*)$' to specify the key name for the parsed field",
		)
	}

	v, err := c.Value.(string)
	if err != nil {
		return nil, fmt.Errorf("invalid replacement string: %s", err)
	}

	replaceOperator := &replaceOperator{
                TransformerOperator: transformerOperator
                Regex:               r
                Value:		     v
        }

	return []operator.Operator{replaceOperator}, nil
}

// ReplaceOperator is an operator that performs regex replacement in an entry.
type ReplaceOperator struct {
	helper.TransformerOperator
	regexp *regexp.Regexp
	Value  interface{}
}

// Process will parse an entry for regex.
func (r *ReplaceOperator) Process(ctx context.Context, entry *entry.Entry) error {
	return r.ProcessWith(ctx, entry, r.Transform)
}

// Transform will parse a value using the supplied regex and perform
// the appropriate replacements
func (r *ReplaceOperator) Transform(value interface{}) (error) {
	var matches []string
	switch m := value.(type) {
	case string:
		matches = r.regexp.ReplaceAllString(m, r.Value)
		if matches == nil {
			return nil, fmt.Errorf("replacement pattern does not match")
		}
	case []byte:
		byteMatches := r.regexp.ReplaceAll(m, []byte(r.Value))
		if byteMatches == nil {
			return nil, fmt.Errorf("replacement pattern does not match")
		}

		matches = make([]string, len(byteMatches))
		for i, byteSlice := range byteMatches {
			matches[i] = string(byteSlice)
		}
	default:
		return nil, fmt.Errorf("type '%T' cannot be parsed as regex in replace", value)
	}

	parsedValues := map[string]interface{}{}
	for i, subexp := range r.regexp.SubexpNames() {
		if i == 0 {
			// Skip whole match
			continue
		}
		if subexp != "" {
			parsedValues[subexp] = matches[i]
		}
	}

	return parsedValues, nil
}
