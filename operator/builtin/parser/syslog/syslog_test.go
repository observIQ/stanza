package syslog

import (
	"context"
	"testing"
	"time"

	"github.com/observiq/stanza/v2/entry"
	"github.com/observiq/stanza/v2/operator"
	"github.com/observiq/stanza/v2/testutil"
	"github.com/stretchr/testify/require"
)

func testLocations() (map[string]*time.Location, error) {
	locations := map[string]string{
		"utc":     "UTC",
		"detroit": "America/Detroit",
		"athens":  "Europe/Athens",
	}

	l := make(map[string]*time.Location)
	for k, v := range locations {
		var err error
		if l[k], err = time.LoadLocation(v); err != nil {
			return nil, err
		}
	}
	return l, nil
}

func TestSyslogParser(t *testing.T) {
	basicConfig := func() *SyslogParserConfig {
		cfg := NewSyslogParserConfig("test_operator_id")
		cfg.OutputIDs = []string{"fake"}
		return cfg
	}

	location, err := testLocations()
	require.NoError(t, err)

	cases := []struct {
		name                 string
		config               *SyslogParserConfig
		inputRecord          interface{}
		expectedTimestamp    time.Time
		expectedRecord       interface{}
		expectedSeverity     entry.Severity
		expectedSeverityText string
	}{
		{
			"RFC3164",
			func() *SyslogParserConfig {
				cfg := basicConfig()
				cfg.Protocol = "rfc3164"
				cfg.Location = location["utc"].String()
				return cfg
			}(),
			"<34>Jan 12 06:30:00 1.2.3.4 apache_server: test message",
			time.Date(time.Now().Year(), 1, 12, 6, 30, 0, 0, location["utc"]),
			map[string]interface{}{
				"appname":  "apache_server",
				"facility": 4,
				"hostname": "1.2.3.4",
				"message":  "test message",
				"priority": 34,
			},
			entry.Critical,
			"crit",
		},
		{
			"RFC3164Detroit",
			func() *SyslogParserConfig {
				cfg := basicConfig()
				cfg.Protocol = "rfc3164"
				cfg.Location = location["detroit"].String()
				return cfg
			}(),
			"<34>Jan 12 06:30:00 1.2.3.4 apache_server: test message",
			time.Date(time.Now().Year(), 1, 12, 6, 30, 0, 0, location["detroit"]),
			map[string]interface{}{
				"appname":  "apache_server",
				"facility": 4,
				"hostname": "1.2.3.4",
				"message":  "test message",
				"priority": 34,
			},
			entry.Critical,
			"crit",
		},
		{
			"RFC3164Athens",
			func() *SyslogParserConfig {
				cfg := basicConfig()
				cfg.Protocol = "rfc3164"
				cfg.Location = location["athens"].String()
				return cfg
			}(),
			"<34>Jan 12 06:30:00 1.2.3.4 apache_server: test message",
			time.Date(time.Now().Year(), 1, 12, 6, 30, 0, 0, location["athens"]),
			map[string]interface{}{
				"appname":  "apache_server",
				"facility": 4,
				"hostname": "1.2.3.4",
				"message":  "test message",
				"priority": 34,
			},
			entry.Critical,
			"crit",
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
			},
			entry.Critical,
			"crit",
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
			entry.Info,
			"info",
		},
		{
			"RFC5424-escaped-r",
			func() *SyslogParserConfig {
				cfg := basicConfig()
				cfg.Protocol = "rfc5424"
				return cfg
			}(),
			`<86>1 2015-08-05T21:58:59.693Z 192.168.2.132 SecureAuth0 23108 ID52020 [SecureAuth@27389 UserHostAddress="192.168.2.132" Realm="SecureAuth0" UserID="Tester2" PEN="27389" user="DOMAIN\\ron"] Found the user for retrieving user's profile`,
			time.Date(2015, 8, 5, 21, 58, 59, 693000000, time.UTC),
			map[string]interface{}{
				"appname":  "SecureAuth0",
				"facility": 10,
				"hostname": "192.168.2.132",
				"message":  "Found the user for retrieving user's profile",
				"msg_id":   "ID52020",
				"priority": 86,
				"proc_id":  "23108",
				"structured_data": map[string]map[string]string{
					"SecureAuth@27389": {
						"PEN":             "27389",
						"Realm":           "SecureAuth0",
						"UserHostAddress": "192.168.2.132",
						"UserID":          "Tester2",
						"user":            "DOMAIN\\\\ron",
					},
				},
				"version": 1,
			},
			entry.Info,
			"info",
		},
		{
			"RFC5424-escaped-t",
			func() *SyslogParserConfig {
				cfg := basicConfig()
				cfg.Protocol = "rfc5424"
				return cfg
			}(),
			`<86>1 2015-08-05T21:58:59.693Z 192.168.2.132 SecureAuth0 23108 ID52020 [SecureAuth@27389 user="DOMAIN\\tom"]`,
			time.Date(2015, 8, 5, 21, 58, 59, 693000000, time.UTC),
			map[string]interface{}{
				"appname":  "SecureAuth0",
				"facility": 10,
				"hostname": "192.168.2.132",
				"msg_id":   "ID52020",
				"priority": 86,
				"proc_id":  "23108",
				"structured_data": map[string]map[string]string{
					"SecureAuth@27389": {
						"user": "DOMAIN\\\\tom",
					},
				},
				"version": 1,
			},
			entry.Info,
			"info",
		},
		{
			"RFC5424-escaped-t",
			func() *SyslogParserConfig {
				cfg := basicConfig()
				cfg.Protocol = "rfc5424"
				return cfg
			}(),
			`<86>1 2015-08-05T21:58:59.693Z 192.168.2.132 SecureAuth0 23108 ID52020 [SecureAuth@27389 user="DOMAIN\\null"]`,
			time.Date(2015, 8, 5, 21, 58, 59, 693000000, time.UTC),
			map[string]interface{}{
				"appname":  "SecureAuth0",
				"facility": 10,
				"hostname": "192.168.2.132",
				"msg_id":   "ID52020",
				"priority": 86,
				"proc_id":  "23108",
				"structured_data": map[string]map[string]string{
					"SecureAuth@27389": {
						"user": "DOMAIN\\\\null",
					},
				},
				"version": 1,
			},
			entry.Info,
			"info",
		},
		{
			"RFC5424-escaped-quote",
			func() *SyslogParserConfig {
				cfg := basicConfig()
				cfg.Protocol = "rfc5424"
				return cfg
			}(),
			`<86>1 2015-08-05T21:58:59.693Z 192.168.2.132 SecureAuth0 23108 ID52020 [SecureAuth@27389 user="invalid user: \"null\""]`,
			time.Date(2015, 8, 5, 21, 58, 59, 693000000, time.UTC),
			map[string]interface{}{
				"appname":  "SecureAuth0",
				"facility": 10,
				"hostname": "192.168.2.132",
				"msg_id":   "ID52020",
				"priority": 86,
				"proc_id":  "23108",
				"structured_data": map[string]map[string]string{
					"SecureAuth@27389": {
						"user": "invalid user: \"null\"",
					},
				},
				"version": 1,
			},
			entry.Info,
			"info",
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
				"structured_data": map[string]map[string]string{
					"verylongsdnamethatisgreaterthan32bytes@12345": {
						"UserHostAddress": "192.168.2.132",
					},
				},
				"version": 1,
			},
			entry.Info,
			"info",
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
				require.Equal(t, tc.expectedRecord, e.Record)
				require.Equal(t, tc.expectedTimestamp, e.Timestamp)
				require.Equal(t, tc.expectedSeverity, e.Severity)
				require.Equal(t, tc.expectedSeverityText, e.SeverityText)
			case <-time.After(time.Second):
				require.FailNow(t, "Timed out waiting for entry to be processed")
			}
		})
	}
}

func TestHandleSymbols(t *testing.T) {
	cases := []struct {
		name           string
		inputRecord    []byte
		expectedRecord []byte
	}{
		{
			"no-symbols-basic",
			[]byte("basic"),
			[]byte("basic"),
		},
		{
			"no-symbols-quote-escaped",
			[]byte("basic\"quote"),
			[]byte("basic\"quote"),
		},
		{
			"real-newline",
			[]byte("\n"),
			[]byte("\n"),
		},
		{
			"real-return",
			[]byte("\r"),
			[]byte("\r"),
		},
		{
			"real-tab",
			[]byte("\t"),
			[]byte("\t"),
		},
		{
			"symbol-new-line-escaped",
			[]byte("DOMAIN\\nexus"),
			[]byte("DOMAIN\\\\nexus"),
		},
		{
			"symbol-return-escaped",
			[]byte("DOMAIN\\ron"),
			[]byte("DOMAIN\\\\ron"),
		},
		{
			"symbol-tab-escaped",
			[]byte("DOMAIN\\tom"),
			[]byte("DOMAIN\\\\tom"),
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			output := handleSymbols(tc.inputRecord)
			require.Equal(t, tc.expectedRecord, output)
		})
	}
}
