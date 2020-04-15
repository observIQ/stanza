package builtin

import (
	"fmt"
	"regexp"

	"github.com/bluemedora/bplogagent/entry"
	"github.com/bluemedora/bplogagent/plugin"
	"github.com/bluemedora/bplogagent/plugin/helper"
	"go.uber.org/zap"
)

func init() {
	plugin.Register("regex_parser", &RegexParserConfig{})
}

// RegexParserConfig is the configuration of a regex parser plugin.
type RegexParserConfig struct {
	helper.BasicPluginConfig      `mapstructure:",squash" yaml:",inline"`
	helper.BasicTransformerConfig `mapstructure:",squash" yaml:",inline"`

	// TODO design these params better
	Field            *entry.FieldSelector
	DestinationField *entry.FieldSelector
	Regex            string
}

// Build will build a regex parser plugin.
func (c RegexParserConfig) Build(context plugin.BuildContext) (plugin.Plugin, error) {
	basicPlugin, err := c.BasicPluginConfig.Build(context.Logger)
	if err != nil {
		return nil, err
	}

	basicTransformer, err := c.BasicTransformerConfig.Build()
	if err != nil {
		return nil, err
	}

	if c.Field == nil {
		var fs entry.FieldSelector = entry.FieldSelector([]string{})
		c.Field = &fs
	}

	if c.DestinationField == nil {
		c.DestinationField = c.Field
	}

	if c.Regex == "" {
		return nil, fmt.Errorf("missing required field 'regex'")
	}

	r, err := regexp.Compile(c.Regex)
	if err != nil {
		return nil, fmt.Errorf("compiling regex: %s", err)
	}

	regexParser := &RegexParser{
		BasicPlugin:      basicPlugin,
		BasicTransformer: basicTransformer,

		field:            *c.Field,
		destinationField: *c.DestinationField,
		regexp:           r,
	}

	return regexParser, nil
}

// RegexParser is a plugin that parses regex in an entry.
type RegexParser struct {
	helper.BasicPlugin
	helper.BasicLifecycle
	helper.BasicTransformer

	field            entry.FieldSelector
	destinationField entry.FieldSelector
	regexp           *regexp.Regexp
}

// Process will parse a field in the entry as regex
func (p *RegexParser) Process(entry *entry.Entry) error {
	newEntry, err := p.parse(entry)
	if err != nil {
		p.Warnw("Failed to parse as regex", zap.Error(err))
		return p.Output.Process(entry)
	}

	return p.Output.Process(newEntry)
}

func (p *RegexParser) parse(entry *entry.Entry) (*entry.Entry, error) {
	message, ok := entry.Get(p.field)
	if !ok {
		return nil, fmt.Errorf("field %s does not exist on the record", p.field)
	}

	var matches []string
	switch m := message.(type) {
	case string:
		matches = p.regexp.FindStringSubmatch(m)
		if matches == nil {
			return nil, fmt.Errorf("regex pattern does not match value '%s'", m)
		}
	case []byte:
		byteMatches := p.regexp.FindSubmatch(m)
		if byteMatches == nil {
			return nil, fmt.Errorf("regex pattern does not match value '%s'", m)
		}

		matches = make([]string, 0, len(byteMatches))
		for i, byteSlice := range byteMatches {
			matches[i] = string(byteSlice)
		}
	default:
		return nil, fmt.Errorf("field %s can not be parsed with regex because it is of type %T", p.field, message)
	}

	newFields := map[string]interface{}{}
	for i, subexp := range p.regexp.SubexpNames() {
		if i == 0 {
			// Skip whole match
			continue
		}
    if subexp != "" {
      newFields[subexp] = matches[i]
    }
	}

	entry.Set(p.destinationField, newFields)

	return entry, nil
}
