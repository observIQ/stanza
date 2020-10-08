package syslog

import (
	"context"
	"testing"
	"time"

	"github.com/observiq/stanza/entry"
	"github.com/observiq/stanza/operator"
	"github.com/observiq/stanza/testutil"
	"github.com/stretchr/testify/require"
)

func TestSyslogParser(t *testing.T) {
	basicConfig := func() *SyslogParserConfig {
		cfg := NewSyslogParserConfig("test_operator_id")
		cfg.OutputIDs = []string{"fake"}
		return cfg
	}

	cases := []struct {
		name              string
		config            *SyslogParserConfig
		inputRecord       interface{}
		expectedTimestamp time.Time
		expectedRecord    interface{}
	}{
		{
			"RFC3164",
			func() *SyslogParserConfig {
				cfg := basicConfig()
				cfg.Protocol = "rfc3164"
				return cfg
			}(),
			"<34>Jan 12 06:30:00 1.2.3.4 apache_server: test message",
			time.Date(time.Now().Year(), 1, 12, 6, 30, 0, 0, time.UTC),
			map[string]interface{}{
				"appname":  "apache_server",
				"facility": 4,
				"hostname": "1.2.3.4",
				"message":  "test message",
				"priority": 34,
				"severity": 2,
			},
		},
		{
			"RFC3164Bytes",
			func() *SyslogParserConfig {
				cfg := basicConfig()
				cfg.Protocol = "rfc3164"
				return cfg
			}(),
			[]byte("<34>Jan 12 06:30:00 1.2.3.4 apache_server: test message"),
			time.Date(time.Now().Year(), 1, 12, 6, 30, 0, 0, time.UTC),
			map[string]interface{}{
				"appname":  "apache_server",
				"facility": 4,
				"hostname": "1.2.3.4",
				"message":  "test message",
				"priority": 34,
				"severity": 2,
			},
		},
		{
			"RFC5424",
			func() *SyslogParserConfig {
				cfg := basicConfig()
				cfg.Protocol = "rfc5424"
				return cfg
			}(),
			`<86>1 2015-08-05T21:58:59.693Z 192.168.2.132 SecureAuth0 23108 ID52020 [SecureAuth@27389 UserHostAddress="192.168.2.132" Realm="SecureAuth0" UserID="Tester2" PEN="27389"] Found the user for retrieving user's profile`,
			time.Date(2015, 8, 5, 21, 58, 59, 693000000, time.UTC),
			map[string]interface{}{
				"appname":  "SecureAuth0",
				"facility": 10,
				"hostname": "192.168.2.132",
				"message":  "Found the user for retrieving user's profile",
				"msg_id":   "ID52020",
				"priority": 86,
				"proc_id":  "23108",
				"severity": 6,
				"structured_data": map[string]map[string]string{
					"SecureAuth@27389": {
						"PEN":             "27389",
						"Realm":           "SecureAuth0",
						"UserHostAddress": "192.168.2.132",
						"UserID":          "Tester2",
					},
				},
				"version": 1,
			},
		},
		{
			"RFC5424LongSDName",
			func() *SyslogParserConfig {
				cfg := basicConfig()
				cfg.Protocol = "rfc5424"
				return cfg
			}(),
			`<86>1 2015-08-05T21:58:59.693Z 192.168.2.132 SecureAuth0 23108 ID52020 [verylongsdnamethatisgreaterthan32bytes@12345 UserHostAddress="192.168.2.132"] my message`,
			time.Date(2015, 8, 5, 21, 58, 59, 693000000, time.UTC),
			map[string]interface{}{
				"appname":  "SecureAuth0",
				"facility": 10,
				"hostname": "192.168.2.132",
				"message":  "my message",
				"msg_id":   "ID52020",
				"priority": 86,
				"proc_id":  "23108",
				"severity": 6,
				"structured_data": map[string]map[string]string{
					"verylongsdnamethatisgreaterthan32bytes@12345": {
						"UserHostAddress": "192.168.2.132",
					},
				},
				"version": 1,
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ops, err := tc.config.Build(testutil.NewBuildContext(t))
			op := ops[0]
			require.NoError(t, err)

			fake := testutil.NewFakeOutput(t)
			err = op.SetOutputs([]operator.Operator{fake})
			require.NoError(t, err)

			newEntry := entry.New()
			newEntry.Record = tc.inputRecord
			err = op.Process(context.Background(), newEntry)
			require.NoError(t, err)

			select {
			case e := <-fake.Received:
				require.Equal(t, e.Record, tc.expectedRecord)
				require.Equal(t, tc.expectedTimestamp, e.Timestamp)
			case <-time.After(time.Second):
				require.FailNow(t, "Timed out waiting for entry to be processed")
			}
		})
	}
}
