package builtin

import (
	"fmt"

	"github.com/bluemedora/bplogagent/entry"
	"github.com/bluemedora/bplogagent/plugin"
	"github.com/bluemedora/bplogagent/plugin/helper"
	"github.com/influxdata/go-syslog/v3"
	"github.com/influxdata/go-syslog/v3/rfc3164"
	"github.com/influxdata/go-syslog/v3/rfc5424"
)

func init() {
	plugin.Register("syslog_parser", &SyslogParserConfig{})
}

// SyslogParserConfig is the configuration of a syslog parser plugin.
type SyslogParserConfig struct {
	helper.BasicPluginConfig      `mapstructure:",squash" yaml:",inline"`
	helper.BasicTransformerConfig `mapstructure:",squash" yaml:",inline"`

	Field            *entry.FieldSelector
	DestinationField *entry.FieldSelector
	Protocol         string `yaml:",omitempty"`
}

// Build will build a JSON parser plugin.
func (c SyslogParserConfig) Build(context plugin.BuildContext) (plugin.Plugin, error) {
	basicPlugin, err := c.BasicPluginConfig.Build(context.Logger)
	if err != nil {
		return nil, err
	}

	basicTransformer, err := c.BasicTransformerConfig.Build()
	if err != nil {
		return nil, err
	}

	if c.Field == nil {
		var fs entry.FieldSelector = entry.SingleFieldSelector([]string{})
		c.Field = &fs
	}

	if c.DestinationField == nil {
		*c.DestinationField = *c.Field
	}

	if c.Protocol == "" {
		return nil, fmt.Errorf("missing field 'protocol'")
	}

	machine, err := c.buildMachine()
	if err != nil {
		return nil, err
	}

	syslogParser := &SyslogParser{
		BasicPlugin:      basicPlugin,
		BasicTransformer: basicTransformer,

		field:            *c.Field,
		destinationField: *c.DestinationField,
		machine:          machine,
	}

	return syslogParser, nil
}

func (c SyslogParserConfig) buildMachine() (syslog.Machine, error) {
	switch c.Protocol {
	case "rfc3164":
		return rfc3164.NewMachine(), nil
	case "rfc5424":
		return rfc5424.NewMachine(), nil
	default:
		return nil, fmt.Errorf("invalid protocol %s", c.Protocol)
	}
}

// SyslogParser is a plugin that parses syslog.
type SyslogParser struct {
	helper.BasicPlugin
	helper.BasicLifecycle
	helper.BasicTransformer

	field            entry.FieldSelector
	destinationField entry.FieldSelector
	machine          syslog.Machine
}

// Process will parse an entry field as syslog.
func (s *SyslogParser) Process(entry *entry.Entry) error {
	newEntry, err := s.parse(entry)
	if err != nil {
		return err
	}

	return s.Output.Process(newEntry)
}

// parse will parse an entry.
func (s *SyslogParser) parse(entry *entry.Entry) (*entry.Entry, error) {
	bytes, err := s.bytesFromField(entry, s.field)
	if err != nil {
		return nil, err
	}

	parsedValues, err := s.parseAsMap(bytes)
	if err != nil {
		return nil, err
	}

	entry.Set(s.destinationField, parsedValues)
	return entry, nil
}

func (s *SyslogParser) bytesFromField(entry *entry.Entry, field entry.FieldSelector) ([]byte, error) {
	value, ok := entry.Get(field)
	if !ok {
		return nil, fmt.Errorf("field '%s' does not exist on the entry", field)
	}

	switch v := value.(type) {
	case string:
		return []byte(v), nil
	case []byte:
		return v, nil
	default:
		return nil, fmt.Errorf("unable to parse field '%v' of type '%T'", field, value)
	}
}

func (s *SyslogParser) parseAsMap(bytes []byte) (map[string]interface{}, error) {
	parsedValue, err := s.machine.Parse(bytes)
	if err != nil {
		return nil, fmt.Errorf("failed to parse as syslog: %s", err)
	}

	if syslogMessage, ok := parsedValue.(*rfc3164.SyslogMessage); ok {
		return map[string]interface{}{
			"timestamp": syslogMessage.Timestamp,
			"priority":  syslogMessage.Priority,
			"facility":  syslogMessage.Facility,
			"severity":  syslogMessage.Severity,
			"hostname":  syslogMessage.Hostname,
			"appname":   syslogMessage.Appname,
			"proc_id":   syslogMessage.ProcID,
			"msg_id":    syslogMessage.MsgID,
			"message":   syslogMessage.Message,
		}, nil
	}

	if syslogMessage, ok := parsedValue.(*rfc5424.SyslogMessage); ok {
		return map[string]interface{}{
			"timestamp":       syslogMessage.Timestamp,
			"priority":        syslogMessage.Priority,
			"facility":        syslogMessage.Facility,
			"severity":        syslogMessage.Severity,
			"hostname":        syslogMessage.Hostname,
			"appname":         syslogMessage.Appname,
			"proc_id":         syslogMessage.ProcID,
			"msg_id":          syslogMessage.MsgID,
			"message":         syslogMessage.Message,
			"structured_data": syslogMessage.StructuredData,
			"version":         syslogMessage.Version,
		}, nil
	}

	return nil, fmt.Errorf("parsed value was not rfc3164 or rfc5424 compliant")
}
