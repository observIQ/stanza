package helper

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/bluemedora/bplogagent/entry"
	"github.com/bluemedora/bplogagent/plugin"
)

const minSeverity = 0
const maxSeverity = 100

// map[string or int input]sev-level
func getBuiltinMapping(name string) severityMap {
	switch name {
	case "none":
		return map[string]entry.Severity{}
	case "aliases":
		return map[string]entry.Severity{
			"default":     entry.Default,
			"trace":       entry.Trace,
			"debug":       entry.Debug,
			"info":        entry.Info,
			"notice":      entry.Notice,
			"warning":     entry.Warning,
			"error":       entry.Error,
			"critical":    entry.Critical,
			"alert":       entry.Alert,
			"emergency":   entry.Emergency,
			"catastrophe": entry.Catastrophe,
		}
	default:
		mapping := getBuiltinMapping("aliases")
		mapping.add(entry.Warning, "warn")
		mapping.add(entry.Error, "err")
		mapping.add(entry.Critical, "crit")
		return mapping
	}
}

func (s severityMap) add(severity entry.Severity, parseableValues ...string) {
	for _, str := range parseableValues {
		s[str] = severity
	}
}

const (
	// HTTP2xx is a special key that is represents a range from 200 to 299. Literal value is "2xx"
	HTTP2xx = "2xx"

	// HTTP3xx is a special key that is represents a range from 300 to 399. Literal value is "3xx"
	HTTP3xx = "3xx"

	// HTTP4xx is a special key that is represents a range from 400 to 499. Literal value is "4xx"
	HTTP4xx = "4xx"

	// HTTP5xx is a special key that is represents a range from 500 to 599. Literal value is "5xx"
	HTTP5xx = "5xx"
)

// SeverityParserConfig allows users to specify how to parse a severity from a field.
type SeverityParserConfig struct {
	ParseFrom entry.Field                 `json:"parse_from,omitempty" yaml:"parse_from,omitempty"`
	Preserve  bool                        `json:"preserve"   yaml:"preserve"`
	Preset    string                      `json:"preset" yaml:"preset"`
	Mapping   map[interface{}]interface{} `json:"mapping"   yaml:"mapping"`
}

// Build builds a SeverityParser from a SeverityParserConfig
func (c *SeverityParserConfig) Build(context plugin.BuildContext) (SeverityParser, error) {

	pluginMapping := getBuiltinMapping(c.Preset)

	for severity, unknown := range c.Mapping {
		sev, err := validateSeverity(severity)
		if err != nil {
			return SeverityParser{}, err
		}

		switch u := unknown.(type) {
		case []interface{}: // check before interface{}
			for _, value := range u {
				v, err := parseableValues(value)
				if err != nil {
					return SeverityParser{}, err
				}
				pluginMapping.add(sev, v...)
			}
		case interface{}:
			v, err := parseableValues(u)
			if err != nil {
				return SeverityParser{}, err
			}
			pluginMapping.add(sev, v...)
		}
	}

	p := SeverityParser{
		ParseFrom: c.ParseFrom,
		Preserve:  c.Preserve,
		Mapping:   pluginMapping,
	}

	return p, nil
}

func validateSeverity(severity interface{}) (entry.Severity, error) {
	if sev, err := getBuiltinMapping("aliases").find(severity); err != nil {
		return entry.Nil, err
	} else if sev != entry.Nil {
		return sev, nil
	}

	// If integer between 0 and 100
	var intSev int
	switch s := severity.(type) {
	case int:
		intSev = s
	case string:
		i, err := strconv.ParseInt(s, 10, 8)
		if err != nil {
			return entry.Nil, fmt.Errorf("%s cannot be used as a severity", severity)
		}
		intSev = int(i)
	default:
		return entry.Nil, fmt.Errorf("type %T cannot be used as a severity (%v)", severity, severity)
	}

	if intSev < minSeverity || intSev > maxSeverity {
		return entry.Nil, fmt.Errorf("severity must be between %d and %d", minSeverity, maxSeverity)
	}
	return entry.Severity(intSev), nil
}

func isRange(value interface{}) (int, int, bool) {
	rawMap, ok := value.(map[interface{}]interface{})
	if !ok {
		return 0, 0, false
	}

	min, minOK := rawMap["min"]
	max, maxOK := rawMap["max"]
	if !minOK || !maxOK {
		return 0, 0, false
	}

	minInt, minOK := min.(int)
	maxInt, maxOK := max.(int)
	if !minOK || !maxOK {
		return 0, 0, false
	}

	return minInt, maxInt, true
}

func expandRange(min, max int) []string {
	if min > max {
		min, max = max, min
	}

	rangeOfStrings := []string{}
	for i := min; i <= max; i++ {
		rangeOfStrings = append(rangeOfStrings, strconv.Itoa(i))
	}
	return rangeOfStrings
}

func parseableValues(value interface{}) ([]string, error) {
	switch v := value.(type) {
	case int:
		return []string{strconv.Itoa(v)}, nil // store as string because we will compare as string
	case string:
		switch v {
		case HTTP2xx:
			return expandRange(200, 299), nil
		case HTTP3xx:
			return expandRange(300, 399), nil
		case HTTP4xx:
			return expandRange(400, 499), nil
		case HTTP5xx:
			return expandRange(500, 599), nil
		default:
			return []string{strings.ToLower(v)}, nil
		}
	case []byte:
		return []string{strings.ToLower(string(v))}, nil
	default:
		min, max, ok := isRange(v)
		if ok {
			return expandRange(min, max), nil
		}
		return nil, fmt.Errorf("type %T cannot be parsed as a severity", v)
	}
}
