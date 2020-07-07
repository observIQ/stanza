package parser

import (
	"context"
	"fmt"
	"regexp"

	"github.com/observiq/bplogagent/entry"
	"github.com/observiq/bplogagent/plugin"
	"github.com/observiq/bplogagent/plugin/helper"
)

func init() {
	plugin.Register("regex_parser", &RegexParserConfig{})
}

// RegexParserConfig is the configuration of a regex parser plugin.
type RegexParserConfig struct {
	helper.ParserConfig `yaml:",inline"`

	Regex string `json:"regex" yaml:"regex"`
}

// Build will build a regex parser plugin.
func (c RegexParserConfig) Build(context plugin.BuildContext) (plugin.Plugin, error) {
	parserPlugin, err := c.ParserConfig.Build(context)
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

	regexParser := &RegexParser{
		ParserPlugin: parserPlugin,
		regexp:       r,
	}

	return regexParser, nil
}

// RegexParser is a plugin that parses regex in an entry.
type RegexParser struct {
	helper.ParserPlugin
	regexp *regexp.Regexp
}

// Process will parse an entry for regex.
func (r *RegexParser) Process(ctx context.Context, entry *entry.Entry) error {
	return r.ParserPlugin.ProcessWith(ctx, entry, r.parse)
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
