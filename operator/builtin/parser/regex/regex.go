package regex

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
	var raw string
	switch m := value.(type) {
	case string:
		raw = m
	case []byte:
		raw = string(m)
	default:
		return nil, fmt.Errorf("type '%T' cannot be parsed as regex", value)
	}
	return r.match(raw)
}

func (r *RegexParser) match(value string) (interface{}, error) {
	if r.Cache != nil {
		if cacheResult, ok := r.Cache.Get(value); ok {
			return cacheResult, nil
		}
	}

	matches := r.regexp.FindStringSubmatch(value)
	if matches == nil {
		return nil, fmt.Errorf("regex pattern does not match")
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

	if r.Cache != nil {
		r.Cache.Add(value, parsedValues)
		r.Debugf("created cached entry: %s", value)
	}

	return parsedValues, nil
}
