package googlecloud

import (
	"github.com/observiq/stanza/v2/entry"
	sev "google.golang.org/genproto/googleapis/logging/type"
)

var fastSev = map[entry.Severity]sev.LogSeverity{
	entry.Catastrophe: sev.LogSeverity_EMERGENCY,
	entry.Emergency:   sev.LogSeverity_EMERGENCY,
	entry.Alert:       sev.LogSeverity_ALERT,
	entry.Critical:    sev.LogSeverity_CRITICAL,
	entry.Error:       sev.LogSeverity_ERROR,
	entry.Warning:     sev.LogSeverity_WARNING,
	entry.Notice:      sev.LogSeverity_NOTICE,
	entry.Info:        sev.LogSeverity_INFO,
	entry.Debug:       sev.LogSeverity_DEBUG,
	entry.Trace:       sev.LogSeverity_DEBUG,
	entry.Default:     sev.LogSeverity_DEFAULT,
}

func convertSeverity(s entry.Severity) sev.LogSeverity {
	if logSev, ok := fastSev[s]; ok {
		return logSev
	}

	switch {
	case s >= entry.Emergency:
		return sev.LogSeverity_EMERGENCY
	case s >= entry.Alert:
		return sev.LogSeverity_ALERT
	case s >= entry.Critical:
		return sev.LogSeverity_CRITICAL
	case s >= entry.Error:
		return sev.LogSeverity_ERROR
	case s >= entry.Warning:
		return sev.LogSeverity_WARNING
	case s >= entry.Notice:
		return sev.LogSeverity_NOTICE
	case s >= entry.Info:
		return sev.LogSeverity_INFO
	case s > entry.Default:
		return sev.LogSeverity_DEBUG
	default:
		return sev.LogSeverity_DEFAULT
	}
}
