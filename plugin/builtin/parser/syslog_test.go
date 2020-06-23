package parser

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/bluemedora/bplogagent/entry"
	"github.com/bluemedora/bplogagent/internal/testutil"
	"github.com/bluemedora/bplogagent/plugin"
	"github.com/bluemedora/bplogagent/plugin/helper"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestSyslogParser(t *testing.T) {
	basicConfig := func() *SyslogParserConfig {
		return &SyslogParserConfig{
			ParserConfig: helper.ParserConfig{
				TransformerConfig: helper.TransformerConfig{
					BasicConfig: helper.BasicConfig{
						PluginID:   "test_plugin_id",
						PluginType: "syslog_parser",
					},
					OutputID: "output1",
				},
			},
		}
	}

	times := map[string]time.Time{
		"RFC3164": func() time.Time {
			t, _ := time.Parse("2006-01-02 15:04:05.999999999 -0700 MST", fmt.Sprintf("%d-01-12 06:30:00 +0000 UTC", time.Now().Year()))
			return t
		}(),
		"RFC5424": func() time.Time {
			t, _ := time.Parse("2006-01-02 15:04:05.999999999 -0700 MST", "2015-08-05 21:58:59.693 +0000 UTC")
			return t
		}(),
	}

	cases := []struct {
		name           string
		config         *SyslogParserConfig
		inputRecord    interface{}
		expectedRecord interface{}
	}{
		{
			"RFC3164",
			func() *SyslogParserConfig {
				cfg := basicConfig()
				cfg.Protocol = "rfc3164"
				return cfg
			}(),
			"<34>Jan 12 06:30:00 1.2.3.4 apache_server: test message",
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
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			buildContext := testutil.NewBuildContext(t)
			newPlugin, err := tc.config.Build(buildContext)
			require.NoError(t, err)
			syslogParser := newPlugin.(*SyslogParser)

			mockOutput := testutil.NewMockPlugin("output1")
			entryChan := make(chan *entry.Entry, 1)
			mockOutput.On("Process", mock.Anything, mock.Anything).Run(func(args mock.Arguments) {
				entryChan <- args.Get(1).(*entry.Entry)
			}).Return(nil)

			err = syslogParser.SetOutputs([]plugin.Plugin{mockOutput})
			require.NoError(t, err)

			newEntry := entry.New()
			newEntry.Record = tc.inputRecord
			err = syslogParser.Process(context.Background(), newEntry)
			require.NoError(t, err)

			select {
			case e := <-entryChan:
				require.Equal(t, e.Record, tc.expectedRecord)
				require.Equal(t, e.Timestamp, times[tc.name])
			case <-time.After(time.Second):
				require.FailNow(t, "Timed out waiting for entry to be processed")
			}
		})
	}
}

func Test_setTimestampYear(t *testing.T) {
	t.Run("Normal", func(t *testing.T) {
		now = func() time.Time {
			return time.Date(2020, 06, 16, 3, 31, 34, 525, time.UTC)
		}

		noYear := time.Date(0, 06, 16, 3, 31, 34, 525, time.UTC)
		yearAdded := setTimestampYear(&noYear)
		expected := time.Date(2020, 06, 16, 3, 31, 34, 525, time.UTC)
		require.Equal(t, &expected, yearAdded)
	})

	t.Run("FutureOneDay", func(t *testing.T) {
		now = func() time.Time {
			return time.Date(2020, 01, 16, 3, 31, 34, 525, time.UTC)
		}

		noYear := time.Date(0, 01, 17, 3, 31, 34, 525, time.UTC)
		yearAdded := setTimestampYear(&noYear)
		expected := time.Date(2020, 01, 17, 3, 31, 34, 525, time.UTC)
		require.Equal(t, &expected, yearAdded)
	})

	t.Run("FutureEightDays", func(t *testing.T) {
		now = func() time.Time {
			return time.Date(2020, 01, 16, 3, 31, 34, 525, time.UTC)
		}

		noYear := time.Date(0, 01, 24, 3, 31, 34, 525, time.UTC)
		yearAdded := setTimestampYear(&noYear)
		expected := time.Date(2019, 01, 24, 3, 31, 34, 525, time.UTC)
		require.Equal(t, &expected, yearAdded)
	})

	t.Run("RolloverYear", func(t *testing.T) {
		now = func() time.Time {
			return time.Date(2020, 01, 01, 3, 31, 34, 525, time.UTC)
		}

		noYear := time.Date(0, 12, 31, 3, 31, 34, 525, time.UTC)
		yearAdded := setTimestampYear(&noYear)
		expected := time.Date(2019, 12, 31, 3, 31, 34, 525, time.UTC)
		require.Equal(t, &expected, yearAdded)
	})
}
