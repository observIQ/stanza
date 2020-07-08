package helper

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/observiq/carbon/entry"
	"github.com/observiq/carbon/errors"
)

// SeverityParser is a helper that parses severity onto an entry.
type SeverityParser struct {
	ParseFrom entry.Field
	Preserve  bool
	Mapping   severityMap
}

// Parse will parse severity from a field and attach it to the entry
func (p *SeverityParser) Parse(ctx context.Context, ent *entry.Entry) error {
	value, ok := ent.Get(p.ParseFrom)
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
	if severity == entry.Nil {
		severity = entry.Default
	}
	ent.Severity = severity

	if !p.Preserve {
		ent.Delete(p.ParseFrom)
	}

	return nil
}

type severityMap map[string]entry.Severity

func (m severityMap) find(value interface{}) (entry.Severity, error) {
	switch v := value.(type) {
	case int:
		if severity, ok := m[strconv.Itoa(v)]; ok {
			return severity, nil
		}
		return entry.Nil, nil
	case string:
		if severity, ok := m[strings.ToLower(v)]; ok {
			return severity, nil
		}
		return entry.Nil, nil
	case []byte:
		if severity, ok := m[strings.ToLower(string(v))]; ok {
			return severity, nil
		}
		return entry.Nil, nil
	default:
		return entry.Nil, fmt.Errorf("type %T cannot be a severity", v)
	}
}
