package regex

import (
	"context"
	"fmt"
	"regexp"

	"github.com/observiq/stanza/v2/entry"
	"github.com/observiq/stanza/v2/errors"
	"github.com/observiq/stanza/v2/operator"
	"github.com/observiq/stanza/v2/operator/helper"
)

func init() {
	operator.Register("regex_parser", func() operator.Builder { return NewRegexParserConfig("") })
}

// NewRegexParserConfig creates a new regex parser config with default values
func NewRegexParserConfig(operatorID string) *RegexParserConfig {
	return &RegexParserConfig{
		ParserConfig: helper.NewParserConfig(operatorID, "regex_parser"),
	}
}

// RegexParserConfig is the configuration of a regex parser operator.
type RegexParserConfig struct {
	helper.ParserConfig `yaml:",inline"`

	Regex string `json:"regex" yaml:"regex"`
}

// Build will build a regex parser operator.
func (c RegexParserConfig) Build(context operator.BuildContext) ([]operator.Operator, error) {
	parserOperator, err := c.ParserConfig.Build(context)
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

	regexParser := &RegexParser{
		ParserOperator: parserOperator,
		regexp:         r,
	}

	return []operator.Operator{regexParser}, nil
}

// RegexParser is an operator that parses regex in an entry.
type RegexParser struct {
	helper.ParserOperator
	regexp *regexp.Regexp
}

// Process will parse an entry for regex.
func (r *RegexParser) Process(ctx context.Context, entry *entry.Entry) error {
	return r.ParserOperator.ProcessWith(ctx, entry, r.parse)
}

// parse will parse a value using the supplied regex.
func (r *RegexParser) parse(value interface{}) (interface{}, error) {
	var matches []string
	switch m := value.(type) {
	case string:
		matches = r.regexp.FindStringSubmatch(m)
		if matches == nil {
			return nil, fmt.Errorf("regex pattern does not match")
		}
	case []byte:
		byteMatches := r.regexp.FindSubmatch(m)
		if byteMatches == nil {
			return nil, fmt.Errorf("regex pattern does not match")
		}

		matches = make([]string, len(byteMatches))
		for i, byteSlice := range byteMatches {
			matches[i] = string(byteSlice)
		}
	default:
		return nil, fmt.Errorf("type '%T' cannot be parsed as regex", value)
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
