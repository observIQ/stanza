package helper

import (
	"context"
	"fmt"

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
func getDefaultMapping() map[interface{}]Severity {
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

// MappingConfig defines how values will be parsed to severity.
type MappingConfig struct {
	Mapping map[interface{}][]interface{} `json:"mapping"   yaml:"mapping"`
}

// SeverityParserConfig allows users to specify how to parse a severity from a field.
type SeverityParserConfig struct {
	ParseFrom     entry.Field `json:"parse_from,omitempty" yaml:"parse_from,omitempty"`
	Preserve      bool        `json:"preserve"   yaml:"preserve"`
	MappingConfig `yaml:",omitempty,inline"`
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

	// used for reference during build
	defaultMapping := getDefaultMapping()

	// used in actual plugin
	pluginMapping := getDefaultMapping()

	for severity, values := range c.Mapping {
		switch s := severity.(type) {
		case string:
			if _, ok := defaultMapping[s]; !ok {
				return SeverityParser{}, fmt.Errorf("Unrecognized severity in mapping: %v", s)
			}
		case []byte:
			if _, ok := defaultMapping[string(s)]; !ok {
				return SeverityParser{}, fmt.Errorf("Unrecognized severity in mapping: %v", s)
			}
		case int:
			if s < minSeverity || s > maxSeverity {
				return SeverityParser{}, fmt.Errorf("Severity must be an integer between %d and %d inclusive", minSeverity, maxSeverity)
			}
		default:
			return SeverityParser{}, fmt.Errorf("type %T cannot be parsed as a severity", s)
		}

		for _, value := range values {
			switch v := value.(type) {
			case string, int:
				pluginMapping[v] = defaultMapping[severity]
			case []byte:
				pluginMapping[string(v)] = defaultMapping[severity]
			default:
				return SeverityParser{}, fmt.Errorf("type %T cannot be parsed as a severity", v)
			}
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
	case string, int:
		if severity, ok := p.Mapping[v]; ok {
			entry.Severity = int(severity)
		}
	case []byte:
		if severity, ok := p.Mapping[string(v)]; ok {
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

// UnmarshalYAML will unmarshal a severity parser config from YAML.
func (c *MappingConfig) UnmarshalYAML(unmarshal func(interface{}) error) error {
	rawMap := make(map[interface{}]interface{})
	err := unmarshal(&rawMap)
	if err != nil {
		return err
	}

	mapping := make(map[interface{}][]interface{})

	for key, value := range rawMap {
		switch v := value.(type) {
		case []interface{}:
			mapping[key] = v
		case interface{}:
			mapping[key] = []interface{}{v}
		}
	}

	c = &MappingConfig{mapping}

	return nil
}
