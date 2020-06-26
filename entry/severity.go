package entry

import (
	"strconv"
)

// Severity indicates the seriousness of a log entry
type Severity int

// ToString converts a severity to a string
func (s Severity) String() string {
	switch s {
	case Default:
		return "default"
	case Trace:
		return "trace"
	case Debug:
		return "debug"
	case Info:
		return "info"
	case Notice:
		return "notice"
	case Warning:
		return "warning"
	case Error:
		return "error"
	case Critical:
		return "critical"
	case Alert:
		return "alert"
	case Emergency:
		return "emergency"
	case Catastrophe:
		return "catastrophe"
	}
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

	// Nil is used to signal that severity is unknown or ambiguous
	Nil Severity = -1
)
