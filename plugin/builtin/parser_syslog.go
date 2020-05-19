package builtin

import (
	"context"
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
	helper.BasicPluginConfig `mapstructure:",squash" yaml:",inline"`
	helper.BasicParserConfig `mapstructure:",squash" yaml:",inline"`

	Protocol string `mapstructure:"protocol" json:"protocol,omitempty" yaml:"protocol,omitempty"`
}

// Build will build a JSON parser plugin.
func (c SyslogParserConfig) Build(context plugin.BuildContext) (plugin.Plugin, error) {
	basicPlugin, err := c.BasicPluginConfig.Build(context.Logger)
	if err != nil {
		return nil, err
	}

	basicParser, err := c.BasicParserConfig.Build(basicPlugin.SugaredLogger)
	if err != nil {
		return nil, err
	}

	if c.Protocol == "" {
		return nil, fmt.Errorf("missing field 'protocol'")
	}

	machine, err := c.buildMachine()
	if err != nil {
		return nil, err
	}

	syslogParser := &SyslogParser{
		BasicPlugin: basicPlugin,
		BasicParser: basicParser,
		machine:     machine,
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
	helper.BasicParser
	machine syslog.Machine
}

// Process will parse an entry field as syslog.
func (s *SyslogParser) Process(ctx context.Context, entry *entry.Entry) error {
	return s.BasicParser.ProcessWith(ctx, entry, s.parse)
}

// parse will parse a value as syslog.
func (s *SyslogParser) parse(value interface{}) (interface{}, error) {
	bytes, err := s.toBytes(value)
	if err != nil {
		return nil, err
	}

	syslog, err := s.machine.Parse(bytes)
	if err != nil {
		return nil, err
	}

	// TODO #172611733 message.Timestamp does not appear to have a year. This should be set to the current year
	if message, ok := syslog.(*rfc3164.SyslogMessage); ok {
		return map[string]interface{}{
			"timestamp": message.Timestamp,
			"priority":  message.Priority,
			"facility":  message.Facility,
			"severity":  message.Severity,
			"hostname":  message.Hostname,
			"appname":   message.Appname,
			"proc_id":   message.ProcID,
			"msg_id":    message.MsgID,
			"message":   message.Message,
		}, nil
	}

	if message, ok := syslog.(*rfc5424.SyslogMessage); ok {
		return map[string]interface{}{
			"timestamp":       message.Timestamp,
			"priority":        message.Priority,
			"facility":        message.Facility,
			"severity":        message.Severity,
			"hostname":        message.Hostname,
			"appname":         message.Appname,
			"proc_id":         message.ProcID,
			"msg_id":          message.MsgID,
			"message":         message.Message,
			"structured_data": message.StructuredData,
			"version":         message.Version,
		}, nil
	}

	return nil, fmt.Errorf("parsed value was not rfc3164 or rfc5424 compliant")
}

func (s *SyslogParser) toBytes(value interface{}) ([]byte, error) {
	switch v := value.(type) {
	case string:
		return []byte(v), nil
	case []byte:
		return v, nil
	default:
		return nil, fmt.Errorf("unable to convert type '%T' to bytes", value)
	}
}
