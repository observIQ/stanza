package builtin

import (
	"context"
	"fmt"
	"time"

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
	helper.BasicPluginConfig `yaml:",inline"`
	helper.BasicParserConfig `yaml:",inline"`

	Protocol string `json:"protocol,omitempty" yaml:"protocol,omitempty"`
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

	machine, err := buildMachine(c.Protocol)
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

func buildMachine(protocol string) (syslog.Machine, error) {
	switch protocol {
	case "rfc3164":
		return rfc3164.NewMachine(), nil
	case "rfc5424":
		return rfc5424.NewMachine(), nil
	default:
		return nil, fmt.Errorf("invalid protocol %s", protocol)
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

	var message map[string]interface{}
	switch m := syslog.(type) {
	case *rfc3164.SyslogMessage:
		message = map[string]interface{}{
			"timestamp": m.Timestamp,
			"priority":  m.Priority,
			"facility":  m.Facility,
			"severity":  m.Severity,
			"hostname":  m.Hostname,
			"appname":   m.Appname,
			"proc_id":   m.ProcID,
			"msg_id":    m.MsgID,
			"message":   m.Message,
		}
	case *rfc5424.SyslogMessage:
		message = map[string]interface{}{
			"timestamp":       m.Timestamp,
			"priority":        m.Priority,
			"facility":        m.Facility,
			"severity":        m.Severity,
			"hostname":        m.Hostname,
			"appname":         m.Appname,
			"proc_id":         m.ProcID,
			"msg_id":          m.MsgID,
			"message":         m.Message,
			"structured_data": m.StructuredData,
			"version":         m.Version,
		}
	default:
		return nil, fmt.Errorf("parsed value was not rfc3164 or rfc5424 compliant")
	}

	// Dereference fields
	for key, val := range message {
		switch v := val.(type) {
		case *string:
			if v == nil {
				delete(message, key)
				continue
			}
			message[key] = *v
		case *uint8:
			if v == nil {
				delete(message, key)
				continue
			}
			message[key] = int(*v)
		case uint16:
			message[key] = int(v)
		case *time.Time:
			if v == nil {
				delete(message, key)
				continue
			}
			message[key] = *v
		case *map[string]map[string]string:
			if v == nil {
				delete(message, key)
				continue
			}
			message[key] = *v
		default:
			return nil, fmt.Errorf("key %s has unknown field of type %T", key, v)
		}
	}

	return message, nil
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
