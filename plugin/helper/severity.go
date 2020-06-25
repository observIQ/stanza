package helper

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/bluemedora/bplogagent/entry"
	"github.com/bluemedora/bplogagent/errors"
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

type severityMap map[string]Severity

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

// SeverityParser is a helper that parses severity onto an entry.
type SeverityParser struct {
	ParseFrom entry.Field
	Preserve  bool
	Mapping   severityMap
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
