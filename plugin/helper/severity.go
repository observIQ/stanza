package helper

import (
	"context"
	"fmt"
	"strings"

	"github.com/bluemedora/bplogagent/entry"
	"github.com/bluemedora/bplogagent/errors"
	"github.com/bluemedora/bplogagent/plugin"
)

// Severity indicates the seriousness of a log entry
type Severity int

const (
	// Default indicates an unknown severity
	Default Severity = 0

	// Trace indicates that the log may be useful for detailed debugging
	Trace = 10

	// Debug indicates that the log may be useful for debugging purposes
	Debug = 20

	// Info indicates that the log may be useful for understanding high level details about an application
	Info = 30

	// Notice indicates that the log should be noticed
	Notice = 40

	// Warning indicates that someone should look into an issue
	Warning = 50

	// Error indicates that something undesireable has actually happened
	Error = 60

	// Critical indicates that a problem requires attention immediately
	Critical = 70

	// Alert indicates that action must be taken immediately
	Alert = 80

	// Emergency indicates that the application is unusable
	Emergency = 90

	// Catastrophe indicates that it is already too late
	Catastrophe = 100
)

const minSeverity = 0
const maxSeverity = 100

// map[string or int input]sev-level
func defaultMapping() map[interface{}]Severity {
	return map[interface{}]Severity{
		int(Default):     Default,
		"default":        Default,
		int(Trace):       Trace,
		"trace":          Trace,
		int(Debug):       Debug,
		"debug":          Debug,
		int(Info):        Info,
		int(Notice):      Notice,
		"notice":         Notice,
		int(Warning):     Warning,
		"warning":        Warning,
		"warn":           Warning,
		int(Error):       Error,
		"error":          Error,
		"err":            Error,
		int(Critical):    Critical,
		"critical":       Critical,
		"crit":           Critical,
		int(Alert):       Alert,
		"alert":          Alert,
		int(Emergency):   Emergency,
		"emergency":      Emergency,
		int(Catastrophe): Catastrophe,
		"catastrophe":    Catastrophe,
	}
}

// SeverityParserConfig allows users to specify how to parse a severity from a field.
type SeverityParserConfig struct {
	ParseFrom entry.Field                 `json:"parse_from,omitempty" yaml:"parse_from,omitempty"`
	Preserve  bool                        `json:"preserve"   yaml:"preserve"`
	Mapping   map[interface{}]interface{} `json:"mapping"   yaml:"mapping"`
}

// SeverityParser is a helper that parses severity onto an entry.
type SeverityParser struct {
	ParseFrom entry.Field
	Preserve  bool

	// map[ValueToParseAsSeverity][Severity]
	Mapping map[interface{}]Severity
}

// Build builds a SeverityParser from a SeverityParserConfig
func (c *SeverityParserConfig) Build(context plugin.BuildContext) (SeverityParser, error) {

	validSeverity := func(severity interface{}) (Severity, error) {

		switch s := severity.(type) {
		case string:
			defaultSev, ok := defaultMapping()[strings.ToLower(s)]
			if !ok {
				return -1, fmt.Errorf("Unrecognized severity in mapping: %v", s)
			}
			return defaultSev, nil
		case []byte:
			defaultSev, ok := defaultMapping()[strings.ToLower(string(s))]
			if !ok {
				return -1, fmt.Errorf("Unrecognized severity in mapping: %v", s)
			}
			return defaultSev, nil
		case int:
			if s < minSeverity || s > maxSeverity {
				return -1, fmt.Errorf("Severity must be an integer between %d and %d inclusive", minSeverity, maxSeverity)
			}
			return Severity(s), nil // may or may not be custom
		default:
			return -1, fmt.Errorf("type %T cannot be parsed as a severity", s)
		}
	}

	validValue := func(value interface{}) (interface{}, error) {
		switch v := value.(type) {
		case int:
			return v, nil
		case string:
			return strings.ToLower(v), nil
		case []byte:
			return strings.ToLower(string(v)), nil
		default:
			return nil, fmt.Errorf("type %T cannot be parsed as a severity", v)
		}
	}

	pluginMapping := defaultMapping()

	for severity, unknown := range c.Mapping {
		sev, err := validSeverity(severity)
		if err != nil {
			return SeverityParser{}, err
		}

		switch u := unknown.(type) {
		case []interface{}:
			for _, value := range u {
				v, err := validValue(value)
				if err != nil {
					return SeverityParser{}, err
				}
				pluginMapping[v] = sev
			}
		case interface{}:
			v, err := validValue(u)
			if err != nil {
				return SeverityParser{}, err
			}
			pluginMapping[v] = sev
		}
	}

	p := SeverityParser{
		ParseFrom: c.ParseFrom,
		Preserve:  c.Preserve,
		Mapping:   pluginMapping,
	}

	return p, nil
}

// Parse will parse severity from a field and attach it to the entry
func (p *SeverityParser) Parse(ctx context.Context, entry *entry.Entry) error {
	value, ok := entry.Get(p.ParseFrom)
	if !ok {
		return errors.NewError(
			"log entry does not have the expected parse_from field",
			"ensure that all entries forwarded to this parser contain the parse_from field",
			"parse_from", p.ParseFrom.String(),
		)
	}

	switch v := value.(type) {
	case int:
		if severity, ok := p.Mapping[v]; ok {
			entry.Severity = int(severity)
		}
	case string:
		if severity, ok := p.Mapping[strings.ToLower(v)]; ok {
			entry.Severity = int(severity)
		}
	case []byte:
		if severity, ok := p.Mapping[strings.ToLower(string(v))]; ok {
			entry.Severity = int(severity)
		}
	default:
		return fmt.Errorf("type %T cannot be parsed as a severity", v)
	}

	if !p.Preserve {
		entry.Delete(p.ParseFrom)
	}

	return nil
}
