package googlecloud

import (
	"testing"

	"github.com/observiq/stanza/entry"
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
			name:             "above emergency",
			severity:         entry.Emergency + 1,
			expectedSeverity: sev.LogSeverity_EMERGENCY,
		},
		{
			name:             "above alert",
			severity:         entry.Alert + 1,
			expectedSeverity: sev.LogSeverity_ALERT,
		},
		{
			name:             "above critical",
			severity:         entry.Critical + 1,
			expectedSeverity: sev.LogSeverity_CRITICAL,
		},
		{
			name:             "above error",
			severity:         entry.Error + 1,
			expectedSeverity: sev.LogSeverity_ERROR,
		},
		{
			name:             "above warning",
			severity:         entry.Warning + 1,
			expectedSeverity: sev.LogSeverity_WARNING,
		},
		{
			name:             "above notice",
			severity:         entry.Notice + 1,
			expectedSeverity: sev.LogSeverity_NOTICE,
		},
		{
			name:             "above info",
			severity:         entry.Info + 1,
			expectedSeverity: sev.LogSeverity_INFO,
		},
		{
			name:             "above debug",
			severity:         entry.Debug + 1,
			expectedSeverity: sev.LogSeverity_DEBUG,
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
