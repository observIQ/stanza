package helper

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/observiq/stanza/entry"
	"github.com/observiq/stanza/errors"
)

// SeverityParser is a helper that parses severity onto an entry.
type SeverityParser struct {
	ParseFrom  entry.Field
	PreserveTo *entry.Field
	Mapping    severityMap
}

// Parse will parse severity from a field and attach it to the entry
func (p *SeverityParser) Parse(ent *entry.Entry) error {
	value, ok := ent.Delete(p.ParseFrom)
	if !ok {
		return errors.NewError(
			"log entry does not have the expected parse_from field",
			"ensure that all entries forwarded to this parser contain the parse_from field",
			"parse_from", p.ParseFrom.String(),
		)
	}

	severity, sevText, err := p.Mapping.find(value)
	if err != nil {
		return errors.Wrap(err, "parse")
	}

	ent.Severity = severity
	ent.SeverityText = sevText

	if p.PreserveTo != nil {
		if err := ent.Set(p.PreserveTo, value); err != nil {
			return errors.Wrap(err, "set preserve_to")
		}
	}

	return nil
}

type severityMap map[string]entry.Severity

// accepts various stringifyable input types and returns
//   1) severity level if found, or default level
//   2) string version of input value
//   3) error if invalid input type
func (m severityMap) find(value interface{}) (entry.Severity, string, error) {
	switch v := value.(type) {
	case int:
		strV := strconv.Itoa(v)
		if severity, ok := m[strV]; ok {
			return severity, strV, nil
		}
		return entry.Default, strV, nil
	case string:
		if severity, ok := m[strings.ToLower(v)]; ok {
			return severity, v, nil
		}
		return entry.Default, v, nil
	case []byte:
		if severity, ok := m[strings.ToLower(string(v))]; ok {
			return severity, string(v), nil
		}
		return entry.Default, string(v), nil
	default:
		return entry.Default, "", fmt.Errorf("type %T cannot be a severity", v)
	}
}
