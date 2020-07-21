package parser

import (
	"context"
	"fmt"
	"time"

	syslog "github.com/influxdata/go-syslog/v3"
	"github.com/influxdata/go-syslog/v3/rfc3164"
	"github.com/influxdata/go-syslog/v3/rfc5424"
	"github.com/observiq/carbon/entry"
	"github.com/observiq/carbon/plugin"
	"github.com/observiq/carbon/plugin/helper"
)

func init() {
	plugin.Register("syslog_parser", func() plugin.Builder { return NewSyslogParserConfig("") })
}

func NewSyslogParserConfig(pluginID string) *SyslogParserConfig {
	return &SyslogParserConfig{
		ParserConfig: helper.NewParserConfig(pluginID, "syslog_parser"),
	}
}

// SyslogParserConfig is the configuration of a syslog parser plugin.
type SyslogParserConfig struct {
	helper.ParserConfig `yaml:",inline"`

	Protocol string `json:"protocol,omitempty" yaml:"protocol,omitempty"`
}

// Build will build a JSON parser plugin.
func (c SyslogParserConfig) Build(context plugin.BuildContext) (plugin.Plugin, error) {
	if c.ParserConfig.TimeParser == nil {
		parseFromField := entry.NewRecordField("timestamp")
		c.ParserConfig.TimeParser = &helper.TimeParser{
			ParseFrom:  &parseFromField,
			LayoutType: helper.NativeKey,
		}
	}

	parserPlugin, err := c.ParserConfig.Build(context)
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
		ParserPlugin: parserPlugin,
		machine:      machine,
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
	helper.ParserPlugin
	machine syslog.Machine
}

// Process will parse an entry field as syslog.
func (s *SyslogParser) Process(ctx context.Context, entry *entry.Entry) error {
	return s.ParserPlugin.ProcessWith(ctx, entry, s.parse)
}

// parse will parse a value as syslog.
func (s *SyslogParser) parse(value interface{}) (interface{}, error) {
	bytes, err := toBytes(value)
	if err != nil {
		return nil, err
	}

	syslog, err := s.machine.Parse(bytes)
	if err != nil {
		return nil, err
	}

	switch message := syslog.(type) {
	case *rfc3164.SyslogMessage:
		return s.parseRFC3164(message)
	case *rfc5424.SyslogMessage:
		return s.parseRFC5424(message)
	default:
		return nil, fmt.Errorf("parsed value was not rfc3164 or rfc5424 compliant")
	}
}

// parseRFC3164 will parse an RFC3164 syslog message.
func (s *SyslogParser) parseRFC3164(syslogMessage *rfc3164.SyslogMessage) (map[string]interface{}, error) {
	value := map[string]interface{}{
		"timestamp": syslogMessage.Timestamp,
		"priority":  syslogMessage.Priority,
		"facility":  syslogMessage.Facility,
		"severity":  syslogMessage.Severity,
		"hostname":  syslogMessage.Hostname,
		"appname":   syslogMessage.Appname,
		"proc_id":   syslogMessage.ProcID,
		"msg_id":    syslogMessage.MsgID,
		"message":   syslogMessage.Message,
	}
	return s.toSafeMap(value)
}

// parseRFC5424 will parse an RFC5424 syslog message.
func (s *SyslogParser) parseRFC5424(syslogMessage *rfc5424.SyslogMessage) (map[string]interface{}, error) {
	value := map[string]interface{}{
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
	}
	return s.toSafeMap(value)
}

// toSafeMap will dereference any pointers on the supplied map.
func (s *SyslogParser) toSafeMap(message map[string]interface{}) (map[string]interface{}, error) {
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

func toBytes(value interface{}) ([]byte, error) {
	switch v := value.(type) {
	case string:
		return []byte(v), nil
	case []byte:
		return v, nil
	default:
		return nil, fmt.Errorf("unable to convert type '%T' to bytes", value)
	}
}
