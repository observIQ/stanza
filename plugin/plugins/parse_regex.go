package plugins

import (
	"fmt"
	"regexp"
	"sync"

	e "github.com/bluemedora/bplogagent/entry"
	pg "github.com/bluemedora/bplogagent/plugin"
)

func init() {
	pg.RegisterConfig("parse_regex", &RegexParserConfig{})
}

type RegexParserConfig struct {
	pg.DefaultPluginConfig    `mapstructure:",squash" yaml:",inline"`
	pg.DefaultOutputterConfig `mapstructure:",squash" yaml:",inline"`
	pg.DefaultInputterConfig  `mapstructure:",squash" yaml:",inline"`

	// TODO design these params better
	Field string
	Regex string
}

func (c RegexParserConfig) Build(context pg.BuildContext) (pg.Plugin, error) {
	defaultPlugin, err := c.DefaultPluginConfig.Build(context.Logger)
	if err != nil {
		return nil, fmt.Errorf("build default plugin: %s", err)
	}

	defaultInputter, err := c.DefaultInputterConfig.Build()
	if err != nil {
		return nil, fmt.Errorf("build default inputter: %s", err)
	}

	defaultOutputter, err := c.DefaultOutputterConfig.Build(context.Plugins)
	if err != nil {
		return nil, fmt.Errorf("build default outputter: %s", err)
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

	plugin := &RegexParser{
		DefaultPlugin:    defaultPlugin,
		DefaultInputter:  defaultInputter,
		DefaultOutputter: defaultOutputter,

		field:  c.Field,
		regexp: r,
	}

	return plugin, nil
}

type RegexParser struct {
	pg.DefaultPlugin
	pg.DefaultOutputter
	pg.DefaultInputter

	field  string
	regexp *regexp.Regexp
}

func (p *RegexParser) Start(wg *sync.WaitGroup) error {
	go func() {
		defer wg.Done()
		for {
			entry, ok := <-p.Input()
			if !ok {
				return
			}

			newEntry, err := p.processEntry(entry)
			if err != nil {
				// TODO better error handling
				p.Warnw("Failed to process entry", "error", err)
				continue
			}

			p.Output() <- newEntry
		}
	}()

	return nil
}

func (p *RegexParser) processEntry(entry e.Entry) (e.Entry, error) {
	message, ok := entry.Record[p.field]
	if !ok {
		return e.Entry{}, fmt.Errorf("field '%s' does not exist on the record", p.field)
	}

	// TODO support bytes?
	messageString, ok := message.(string)
	if !ok {
		return e.Entry{}, fmt.Errorf("field '%s' can not be parsed with regex because it is of type %T", p.field, message)
	}

	matches := p.regexp.FindStringSubmatch(messageString)
	if matches == nil {
		return e.Entry{}, fmt.Errorf("regex pattern does not match field")
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
