package googlecloud

import (
	"github.com/observiq/stanza/entry"
	sev "google.golang.org/genproto/googleapis/logging/type"
)

var fastSev = map[entry.Severity]sev.LogSeverity{
	entry.Fatal4:  sev.LogSeverity_EMERGENCY,
	entry.Fatal:   sev.LogSeverity_EMERGENCY,
	entry.Error3:  sev.LogSeverity_ALERT,
	entry.Error4:  sev.LogSeverity_CRITICAL,
	entry.Error:   sev.LogSeverity_ERROR,
	entry.Warn:    sev.LogSeverity_WARNING,
	entry.Info4:   sev.LogSeverity_NOTICE,
	entry.Info:    sev.LogSeverity_INFO,
	entry.Debug:   sev.LogSeverity_DEBUG,
	entry.Trace:   sev.LogSeverity_DEBUG,
	entry.Default: sev.LogSeverity_DEFAULT,
}

func convertSeverity(s entry.Severity) sev.LogSeverity {
	if logSev, ok := fastSev[s]; ok {
		return logSev
	}

	switch {
	case s >= entry.Fatal:
		return sev.LogSeverity_EMERGENCY
	case s >= entry.Error3:
		return sev.LogSeverity_ALERT
	case s >= entry.Error4:
		return sev.LogSeverity_CRITICAL
	case s >= entry.Error:
		return sev.LogSeverity_ERROR
	case s >= entry.Warn:
		return sev.LogSeverity_WARNING
	case s >= entry.Info4:
		return sev.LogSeverity_NOTICE
	case s >= entry.Info:
		return sev.LogSeverity_INFO
	case s > entry.Default:
		return sev.LogSeverity_DEBUG
	default:
		return sev.LogSeverity_DEFAULT
	}
}
