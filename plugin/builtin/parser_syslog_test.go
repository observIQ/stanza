package builtin

import (
	"context"
	"testing"
	"time"

	"github.com/bluemedora/bplogagent/entry"
	"github.com/bluemedora/bplogagent/plugin"
	"github.com/bluemedora/bplogagent/plugin/helper"
	"github.com/bluemedora/bplogagent/plugin/testutil"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestSyslogParser(t *testing.T) {
	basicConfig := func() *SyslogParserConfig {
		return &SyslogParserConfig{
			BasicPluginConfig: helper.BasicPluginConfig{
				PluginID:   "test_plugin_id",
				PluginType: "syslog_parser",
			},
			BasicParserConfig: helper.BasicParserConfig{
				OutputID: "output1",
			},
		}
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
				"timestamp": func() time.Time {
					t, _ := time.Parse("2006-01-02 15:04:05.999999999 -0700 MST", "0000-01-12 06:30:00 +0000 UTC")
					return t
				}(),
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
				"timestamp": func() time.Time {
					t, _ := time.Parse("2006-01-02 15:04:05.999999999 -0700 MST", "2015-08-05 21:58:59.693 +0000 UTC")
					return t
				}(),
				"version": 1,
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			buildContext := testutil.NewTestBuildContext(t)
			newPlugin, err := tc.config.Build(buildContext)
			require.NoError(t, err)
			syslogParser := newPlugin.(*SyslogParser)

			mockOutput := &testutil.Plugin{}
			mockOutput.On("CanProcess").Return(true)
			mockOutput.On("ID").Return("output1")
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
			case <-time.After(time.Second):
				require.FailNow(t, "Timed out waiting for entry to be processed")
			}
		})
	}
}
