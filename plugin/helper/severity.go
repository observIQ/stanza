package helper

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/bluemedora/bplogagent/entry"
	"github.com/bluemedora/bplogagent/errors"
	"github.com/bluemedora/bplogagent/plugin"
)

// Severity indicates the seriousness of a log entry
type Severity int

// ToString converts a severity to a string
func (s Severity) ToString() string {
	return strconv.Itoa(int(s))
}

const (
	// Default indicates an unknown severity
	Default Severity = 0

	// Trace indicates that the log may be useful for detailed debugging
	Trace Severity = 10

	// Debug indicates that the log may be useful for debugging purposes
	Debug Severity = 20

	// Info indicates that the log may be useful for understanding high level details about an application
	Info Severity = 30

	// Notice indicates that the log should be noticed
	Notice Severity = 40

	// Warning indicates that someone should look into an issue
	Warning Severity = 50

	// Error indicates that something undesireable has actually happened
	Error Severity = 60

	// Critical indicates that a problem requires attention immediately
	Critical Severity = 70

	// Alert indicates that action must be taken immediately
	Alert Severity = 80

	// Emergency indicates that the application is unusable
	Emergency Severity = 90

	// Catastrophe indicates that it is already too late
	Catastrophe Severity = 100

	// used internally
	notFound Severity = -1
)

const minSeverity = 0
const maxSeverity = 100

// map[string or int input]sev-level
func defaultSeverityMap() severityMap {
	return map[string]Severity{
		Default.ToString():     Default,
		"default":              Default,
		Trace.ToString():       Trace,
		"trace":                Trace,
		Debug.ToString():       Debug,
		"debug":                Debug,
		Info.ToString():        Info,
		"info":                 Info,
		Notice.ToString():      Notice,
		"notice":               Notice,
		Warning.ToString():     Warning,
		"warning":              Warning,
		"warn":                 Warning,
		Error.ToString():       Error,
		"error":                Error,
		"err":                  Error,
		Critical.ToString():    Critical,
		"critical":             Critical,
		"crit":                 Critical,
		Alert.ToString():       Alert,
		"alert":                Alert,
		Emergency.ToString():   Emergency,
		"emergency":            Emergency,
		Catastrophe.ToString(): Catastrophe,
		"catastrophe":          Catastrophe,
	}
}

type severityMap map[string]Severity

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
	Mapping   severityMap
}

// Build builds a SeverityParser from a SeverityParserConfig
func (c *SeverityParserConfig) Build(context plugin.BuildContext) (SeverityParser, error) {

	validSeverity := func(severity interface{}) (Severity, error) {
		// If already defined as a standard severity
		if sev, err := defaultSeverityMap().find(severity); err != nil {
			return notFound, err
		} else if sev != notFound {
			return sev, nil
		}

		// If integer between 0 and 100, then allow as custom severity
		if sev, ok := severity.(int); !ok {
			return notFound, fmt.Errorf("type %T cannot be used as a custom severity (%v)", severity, severity)
		} else if sev < minSeverity || sev > maxSeverity {
			return notFound, fmt.Errorf("custom severity must be between %d and %d", minSeverity, maxSeverity)
		} else {
			return Severity(sev), nil
		}
	}

	validValues := func(value interface{}) ([]string, error) {
		switch v := value.(type) {
		case int:
			return []string{strconv.Itoa(v)}, nil // store as string because we will compare as string
		case string:
			return []string{strings.ToLower(v)}, nil
		case []byte:
			return []string{strings.ToLower(string(v))}, nil
		default:
			rawMap, ok := v.(map[interface{}]interface{})
			if !ok {
				return nil, fmt.Errorf("type %T cannot be parsed as a severity", v)
			}

			min, minOK := rawMap["min"]
			max, maxOK := rawMap["max"]
			if !minOK || !maxOK {
				return nil, fmt.Errorf("type %T cannot be parsed as a severity", v)
			}

			minInt, minOK := min.(int)
			maxInt, maxOK := max.(int)
			if !minOK || !maxOK {
				return nil, fmt.Errorf("type %T cannot be parsed as a severity", v)
			}

			if minInt > maxInt {
				minInt, maxInt = maxInt, minInt
			}

			rangeOfStrings := []string{}
			for i := minInt; i <= maxInt; i++ {
				rangeOfStrings = append(rangeOfStrings, strconv.Itoa(i))
			}
			return rangeOfStrings, nil
		}
	}

	pluginMapping := defaultSeverityMap()

	for severity, unknown := range c.Mapping {
		sev, err := validSeverity(severity)
		if err != nil {
			return SeverityParser{}, err
		}

		switch u := unknown.(type) {
		case []interface{}:
			for _, value := range u {
				v, err := validValues(value)
				if err != nil {
					return SeverityParser{}, err
				}
				for _, str := range v {
					pluginMapping[str] = sev
				}
			}
		case interface{}:
			v, err := validValues(u)
			if err != nil {
				return SeverityParser{}, err
			}
			for _, str := range v {
				pluginMapping[str] = sev
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

	severity, err := p.Mapping.find(value)
	if err != nil {
		return errors.Wrap(err, "parse")
	}
	if severity == notFound {
		severity = Default
	}
	entry.Severity = int(severity)

	if !p.Preserve {
		entry.Delete(p.ParseFrom)
	}

	return nil
}

func (m severityMap) find(value interface{}) (Severity, error) {
	switch v := value.(type) {
	case int, Severity:
		if severity, ok := m[fmt.Sprintf("%d", v)]; ok {
			return severity, nil
		}
		return notFound, nil
	case string:
		if severity, ok := m[strings.ToLower(v)]; ok {
			return severity, nil
		}
		return notFound, nil
	case []byte:
		if severity, ok := m[strings.ToLower(string(v))]; ok {
			return severity, nil
		}
		return notFound, nil
	default:
		return notFound, fmt.Errorf("type %T cannot be a severity", v)
	}
}
