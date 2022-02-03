package googlecloud

import (
	"testing"

	"github.com/open-telemetry/opentelemetry-log-collection/entry"
	"github.com/stretchr/testify/require"
	sev "google.golang.org/genproto/googleapis/logging/type"
)

func TestConvertSeverity(t *testing.T) {
	testCases := []struct {
		name             string
		severity         entry.Severity
		expectedSeverity sev.LogSeverity
	}{
		{
			name:             "Default",
			severity:         entry.Default,
			expectedSeverity: sev.LogSeverity_DEFAULT,
		},
		{
			name:             "Trace",
			severity:         entry.Trace,
			expectedSeverity: sev.LogSeverity_DEBUG,
		},
		{
			name:             "Trace2",
			severity:         entry.Trace2,
			expectedSeverity: sev.LogSeverity_DEBUG,
		},
		{
			name:             "Trace3",
			severity:         entry.Trace3,
			expectedSeverity: sev.LogSeverity_DEBUG,
		},
		{
			name:             "Trace4",
			severity:         entry.Trace4,
			expectedSeverity: sev.LogSeverity_DEBUG,
		},
		{
			name:             "Debug",
			severity:         entry.Debug,
			expectedSeverity: sev.LogSeverity_DEBUG,
		},
		{
			name:             "Debug2",
			severity:         entry.Debug2,
			expectedSeverity: sev.LogSeverity_DEBUG,
		},
		{
			name:             "Debug3",
			severity:         entry.Debug3,
			expectedSeverity: sev.LogSeverity_DEBUG,
		},
		{
			name:             "Debug4",
			severity:         entry.Debug4,
			expectedSeverity: sev.LogSeverity_DEBUG,
		},
		{
			name:             "Info",
			severity:         entry.Info,
			expectedSeverity: sev.LogSeverity_INFO,
		},
		{
			name:             "Info2",
			severity:         entry.Info2,
			expectedSeverity: sev.LogSeverity_NOTICE,
		},
		{
			name:             "Info3",
			severity:         entry.Info3,
			expectedSeverity: sev.LogSeverity_NOTICE,
		},
		{
			name:             "Info4",
			severity:         entry.Info4,
			expectedSeverity: sev.LogSeverity_NOTICE,
		},
		{
			name:             "Warn",
			severity:         entry.Warn,
			expectedSeverity: sev.LogSeverity_WARNING,
		},
		{
			name:             "Warn2",
			severity:         entry.Warn2,
			expectedSeverity: sev.LogSeverity_WARNING,
		},
		{
			name:             "Warn3",
			severity:         entry.Warn3,
			expectedSeverity: sev.LogSeverity_WARNING,
		},
		{
			name:             "Warn4",
			severity:         entry.Warn4,
			expectedSeverity: sev.LogSeverity_WARNING,
		},
		{
			name:             "Error",
			severity:         entry.Error,
			expectedSeverity: sev.LogSeverity_ERROR,
		},
		{
			name:             "Error2",
			severity:         entry.Error2,
			expectedSeverity: sev.LogSeverity_CRITICAL,
		},
		{
			name:             "Error3",
			severity:         entry.Error3,
			expectedSeverity: sev.LogSeverity_ALERT,
		},
		{
			name:             "Error4",
			severity:         entry.Error4,
			expectedSeverity: sev.LogSeverity_ALERT,
		},
		{
			name:             "Fatal",
			severity:         entry.Fatal,
			expectedSeverity: sev.LogSeverity_EMERGENCY,
		},
		{
			name:             "Fatal2",
			severity:         entry.Fatal2,
			expectedSeverity: sev.LogSeverity_EMERGENCY,
		},
		{
			name:             "Fatal3",
			severity:         entry.Fatal3,
			expectedSeverity: sev.LogSeverity_EMERGENCY,
		},
		{
			name:             "Fatal4",
			severity:         entry.Fatal4,
			expectedSeverity: sev.LogSeverity_EMERGENCY,
		},

		{
			name:             "unknown",
			severity:         -1,
			expectedSeverity: sev.LogSeverity_DEFAULT,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := convertSeverity(tc.severity)
			require.Equal(t, tc.expectedSeverity, result)
		})
	}
}
