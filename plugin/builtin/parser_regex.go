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

	// TODO design these params better
	Field string
	Regex string
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

	if c.Field == "" {
		return nil, fmt.Errorf("missing required field 'field'")
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

		field:  c.Field,
		regexp: r,
	}

	return regexParser, nil
}

// RegexParser is a plugin that parses regex in an entry.
type RegexParser struct {
	helper.BasicPlugin
	helper.BasicLifecycle
	helper.BasicTransformer

	field  string
	regexp *regexp.Regexp
}

// Process will parse a field in the entry as regex
func (p *RegexParser) Process(entry *entry.Entry) error {
	newEntry, err := p.parse(entry)
	if err != nil {
		// TODO allow continuing with best effort
		return err
	}

	return p.Output.Process(newEntry)
}

func (p *RegexParser) parse(entry *entry.Entry) (*entry.Entry, error) {
	message, ok := entry.Record[p.field]
	if !ok {
		return nil, fmt.Errorf("field '%s' does not exist on the record", p.field)
	}

	// TODO support bytes?
	messageString, ok := message.(string)
	if !ok {
		return nil, fmt.Errorf("field '%s' can not be parsed with regex because it is of type %T", p.field, message)
	}

	matches := p.regexp.FindStringSubmatch(messageString)
	if matches == nil {
		return nil, fmt.Errorf("regex pattern does not match field")
	}

	newFields := map[string]interface{}{}
	for i, subexp := range p.regexp.SubexpNames() {
		if i == 0 {
			// Skip whole match
			continue
		}
		newFields[subexp] = matches[i]
	}

	// TODO allow keeping original message
	// TODO allow flattening fields
	entry.Record[p.field] = newFields

	return entry, nil
}
