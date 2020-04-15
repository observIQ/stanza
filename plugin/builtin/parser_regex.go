package builtin

import (
	"fmt"
	"regexp"

	"github.com/bluemedora/bplogagent/entry"
	"github.com/bluemedora/bplogagent/plugin"
	"github.com/bluemedora/bplogagent/plugin/helper"
)

func init() {
	plugin.Register("regex_parser", &RegexParserConfig{})
}

// RegexParserConfig is the configuration of a regex parser plugin.
type RegexParserConfig struct {
	helper.BasicPluginConfig      `mapstructure:",squash" yaml:",inline"`
	helper.BasicTransformerConfig `mapstructure:",squash" yaml:",inline"`
	helper.BasicParserConfig      `mapstructure:",squash" yaml:",inline"`
	Regex                         string
}

// Build will build a regex parser plugin.
func (c RegexParserConfig) Build(context plugin.BuildContext) (plugin.Plugin, error) {
	basicPlugin, err := c.BasicPluginConfig.Build(context.Logger)
	if err != nil {
		return nil, err
	}

	basicParser, err := c.BasicParserConfig.Build()
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
		BasicPlugin: basicPlugin,
		BasicParser: basicParser,
		regexp:      r,
	}

	return regexParser, nil
}

// RegexParser is a plugin that parses regex in an entry.
type RegexParser struct {
	helper.BasicPlugin
	helper.BasicLifecycle
	helper.BasicParser
	regexp *regexp.Regexp
}

// Process will parse an entry for regex.
func (r *RegexParser) Process(entry *entry.Entry) error {
	return r.BasicParser.ProcessWith(entry, r.parse)
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

		matches = make([]string, 0, len(byteMatches))
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
		parsedValues[subexp] = matches[i]
	}

	return parsedValues, nil
}
