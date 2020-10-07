package windows

import (
	"encoding/json"
	"io/ioutil"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParseSecurity(t *testing.T) {

	testCases := []string{
		"account_name_changed",
		"audit_settings_changed",
		"audit_success",
		"credential_validate_attempt",
		"domain_policy_changed",
		"driver_started",
		"event_processing",
		"local_group_changed",
		"logon",
		"object_added",
		"per_user_audit_policy_table_created",
		"query_blank_password",
		"service_shutdown",
		"service_started",
		"special_logon",
		"time_change",
		"user_account_changed",
		"user_account_created",
		"user_account_enabled",
		"user_added_to_global_group",
		"user_password_reset_attempt",
	}

	for _, tc := range testCases {
		t.Run(tc, func(t *testing.T) {

			testDir := filepath.Join("testdata", "security", tc)
			messageBytes, err := ioutil.ReadFile(filepath.Join(testDir, "message.in"))
			require.NoError(t, err, "problem reading input file")

			message, details := parseSecurity(string(messageBytes))

			// initTestResult(testDir, message, details)

			expectedMessageBytes, err := ioutil.ReadFile(filepath.Join(testDir, "message.out"))
			require.NoError(t, err, "problem reading expected message")
			expectedMessage := string(expectedMessageBytes)

			expectedDetailsBytes, err := ioutil.ReadFile(filepath.Join(testDir, "details.out"))
			require.NoError(t, err, "problem reading expected details")

			// This is a little silly, but if we rely on unmarshaling
			// then []string gets converted to []interface{} and the comparison fails
			detailBytes, err := json.Marshal(details)
			require.NoError(t, err, "problem processing details result")

			require.Equal(t, expectedMessage, message)
			require.JSONEq(t, string(expectedDetailsBytes), string(detailBytes))
		})
	}
}

// Use this to initialize test results from a WEL security message
// make sure to validate manually!
func initTestResult(testDir, message string, details map[string]interface{}) {
	ioutil.WriteFile(filepath.Join(testDir, "message.out"), []byte(message), 0644)
	bytes, _ := json.MarshalIndent(details, "", "  ")
	ioutil.WriteFile(filepath.Join(testDir, "details.out"), bytes, 0644)
}
