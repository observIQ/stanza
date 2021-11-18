package syslog

import (
	"bytes"
	"context"
	"fmt"
	"time"

	sl "github.com/observiq/go-syslog/v3"
	"github.com/observiq/go-syslog/v3/rfc3164"
	"github.com/observiq/go-syslog/v3/rfc5424"
	"github.com/observiq/stanza/v2/operator"
	"github.com/observiq/stanza/v2/operator/helper"
	"github.com/open-telemetry/opentelemetry-log-collection/entry"
)

func init() {
	operator.Register("syslog_parser", func() operator.Builder { return NewSyslogParserConfig("") })
}

// NewSyslogParserConfig creates a new syslog parser config with default values
func NewSyslogParserConfig(operatorID string) *SyslogParserConfig {
	return &SyslogParserConfig{
		ParserConfig: helper.NewParserConfig(operatorID, "syslog_parser"),
	}
}

// SyslogParserConfig is the configuration of a syslog parser operator.
type SyslogParserConfig struct {
	helper.ParserConfig `yaml:",inline"`

	Protocol string `json:"protocol,omitempty" yaml:"protocol,omitempty"`
	Location string `json:"location,omitempty" yaml:"location,omitempty"`
}

// Build will build a JSON parser operator.
func (c SyslogParserConfig) Build(context operator.BuildContext) ([]operator.Operator, error) {
	if c.ParserConfig.TimeParser == nil {
		parseFromField := entry.NewBodyField("timestamp")
		c.ParserConfig.TimeParser = &helper.TimeParser{
			ParseFrom:  &parseFromField,
			LayoutType: helper.NativeKey,
		}
	}

	parserOperator, err := c.ParserConfig.Build(context)
	if err != nil {
		return nil, err
	}

	if c.Protocol == "" {
		return nil, fmt.Errorf("missing field 'protocol'")
	}

	if c.Location == "" {
		c.Location = "UTC"
	}

	location, err := time.LoadLocation(c.Location)
	if err != nil {
		return nil, err
	}

	syslogParser := &SyslogParser{
		ParserOperator: parserOperator,
		protocol:       c.Protocol,
		location:       location,
	}

	return []operator.Operator{syslogParser}, nil
}

func buildMachine(protocol string, location *time.Location) (sl.Machine, error) {
	switch protocol {
	case "rfc3164":
		return rfc3164.NewMachine(rfc3164.WithLocaleTimezone(location)), nil
	case "rfc5424":
		return rfc5424.NewMachine(), nil
	default:
		return nil, fmt.Errorf("invalid protocol %s", protocol)
	}
}

// SyslogParser is an operator that parses syslog.
type SyslogParser struct {
	helper.ParserOperator
	protocol string
	location *time.Location
}

// Process will parse an entry field as syslog.
func (s *SyslogParser) Process(ctx context.Context, entry *entry.Entry) error {
	return s.ParserOperator.ProcessWithCallback(ctx, entry, s.parse, promoteSeverity)
}

// parse will parse a value as syslog.
func (s *SyslogParser) parse(value interface{}) (interface{}, error) {
	b, err := toBytes(value)
	if err != nil {
		return nil, err
	}

	b = handleSymbols(b)

	machine, err := buildMachine(s.protocol, s.location)
	if err != nil {
		return nil, err
	}

	slog, err := machine.Parse(b)
	if err != nil {
		return nil, err
	}

	switch message := slog.(type) {
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

// handleSymbols escapes characters that appear to be symbol characters, but are not.
// If not escaped properly, go-syslog will fail to parse the entry. Quotes that are
// escaped will not be affected.
func handleSymbols(b []byte) []byte {
	b = bytes.Replace(b, []byte(`\`), []byte(`\\`), -1)
	return bytes.Replace(b, []byte(`\\"`), []byte(`\"`), -1)
}

var severityMapping = [...]entry.Severity{
	0: entry.Fatal,
	1: entry.Error3,
	2: entry.Error2,
	3: entry.Error,
	4: entry.Warn,
	5: entry.Info3,
	6: entry.Info,
	7: entry.Debug,
}

var severityText = [...]string{
	0: "emerg",
	1: "alert",
	2: "crit",
	3: "err",
	4: "warning",
	5: "notice",
	6: "info",
	7: "debug",
}

var severityField = entry.NewBodyField("severity")

func promoteSeverity(e *entry.Entry) error {
	sev, ok := severityField.Delete(e)
	if !ok {
		return fmt.Errorf("severity field does not exist")
	}

	sevInt, ok := sev.(int)
	if !ok {
		return fmt.Errorf("severity field is not an int")
	}

	if sevInt < 0 || sevInt > 7 {
		return fmt.Errorf("invalid severity '%d'", sevInt)
	}

	e.Severity = severityMapping[sevInt]
	e.SeverityText = severityText[sevInt]
	return nil
}
