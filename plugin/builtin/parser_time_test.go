package builtin

import (
	"context"
	"testing"
	"time"

	"github.com/bluemedora/bplogagent/entry"
	"github.com/bluemedora/bplogagent/plugin/helper"
	"github.com/bluemedora/bplogagent/plugin/testutil"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func testConfig(flavor, layout string) TimeParserConfig {
	return TimeParserConfig{
		TransformerConfig: helper.TransformerConfig{
			BasicConfig: helper.BasicConfig{
				PluginID:   "test_plugin_id",
				PluginType: "time_parser",
			},
			OutputID: "output1",
		},
		LayoutFlavor: flavor,
		Layout:       layout,
	}
}

func testConfigWithParseFrom(flavor, layout string, parseFrom entry.Field) TimeParserConfig {
	cfg := testConfig(flavor, layout)
	cfg.ParseFrom = parseFrom
	return cfg
}

func TestParser(t *testing.T) {

	testCases := []struct {
		name      string
		cfg       TimeParserConfig
		entry     entry.Entry
		expectErr bool
		result    time.Time
	}{
		{
			name: "gotime-unix",
			cfg:  testConfig("gotime", time.UnixDate),
			entry: entry.Entry{
				Record: "Mon Jan 2 15:04:05 MST 2006",
			},
			expectErr: false,
			result: func() time.Time {
				t, _ := time.Parse(time.UnixDate, "Mon Jan 2 15:04:05 MST 2006")
				return t
			}(),
		},
		{
			name: "gotime-unix-field",
			cfg:  testConfigWithParseFrom("gotime", time.UnixDate, entry.NewField("some_field")),
			entry: entry.Entry{
				Record: map[string]interface{}{
					"some_field": "Mon Jan 2 15:04:05 MST 2006",
				},
			},
			expectErr: false,
			result: func() time.Time {
				t, _ := time.Parse(time.UnixDate, "Mon Jan 2 15:04:05 MST 2006")
				return t
			}(),
		},
		{
			name: "gotime-kitchen",
			cfg:  testConfig("gotime", time.Kitchen),
			entry: entry.Entry{
				Record: "12:34PM",
			},
			expectErr: false,
			result: func() time.Time {
				t, _ := time.Parse(time.Kitchen, "12:34PM")
				return t
			}(),
		},
		{
			name: "gotime-countdown",
			cfg:  testConfig("gotime", "-0700 06 05 04 03 02 01"),
			entry: entry.Entry{
				Record: "-0100 01 01 01 01 01 01",
			},
			expectErr: false,
			result: func() time.Time {
				t, _ := time.Parse("-0700 06 05 04 03 02 01", "-0100 01 01 01 01 01 01")
				return t
			}(),
		},
		{
			name: "gotime-debian-syslog",
			cfg:  testConfig("gotime", "Jan 2 15:04:05"),
			entry: entry.Entry{
				Record: "Jun 11 11:39:45",
			},
			expectErr: false,
			result: func() time.Time {
				t, _ := time.Parse("Jan 2 15:04:05", "Jun 11 11:39:45")
				return t
			}(),
		},
		{
			name: "strptime-debian-syslog",
			cfg:  testConfig("strptime", "%b %e %H:%M:%S"),
			entry: entry.Entry{
				Record: "Jun 11 11:39:45",
			},
			expectErr: false,
			result: func() time.Time {
				t, _ := time.Parse("Jan 2 15:04:05", "Jun 11 11:39:45")
				return t
			}(),
		},
		{
			name: "gotime-pgbouncer",
			cfg:  testConfig("gotime", "2006-01-02 15:04:05.000 MST"),
			entry: entry.Entry{
				Record: "2019-11-05 10:38:35.118 EST",
			},
			expectErr: false,
			result: func() time.Time {
				t, _ := time.Parse("2006-01-02 15:04:05.000 MST", "2019-11-05 10:38:35.118 EST")
				return t
			}(),
		},
		{
			name: "strptime-pgbouncer",
			cfg:  testConfig("strptime", "%Y-%m-%d %H:%M:%S.%L %Z"),
			entry: entry.Entry{
				Record: "2019-11-05 10:38:35.118 EST",
			},
			expectErr: false,
			result: func() time.Time {
				t, _ := time.Parse("2006-01-02 15:04:05.000 MST", "2019-11-05 10:38:35.118 EST")
				return t
			}(),
		},

		// {
		// 	name: "opendistro",
		// 	"2020-06-11T15:39:58,903"
		// },
		// {
		// 	name: "",
		// 	""
		// },
		// {
		// 	name: "",
		// 	""
		// },
		// {
		// 	name: "",
		// 	""
		// },
		// {
		// 	name: "",
		// 	""
		// },
		// {
		// 	name: "",
		// 	""
		// },
		// {
		// 	name: "",
		// 	""
		// },
		{
			name: "strptime-unix",
			cfg:  testConfig("strptime", "%a %b %e %H:%M:%S %Z %Y"),
			entry: entry.Entry{
				Record: "Mon Jan 2 15:04:05 MST 2006",
			},
			expectErr: false,
			result: func() time.Time {
				t, _ := time.Parse(time.UnixDate, "Mon Jan 2 15:04:05 MST 2006")
				return t
			}(),
		},
		{
			name: "strptime-almost-unix",
			cfg:  testConfig("strptime", "%a %b %d %H:%M:%S %Z %Y"),
			entry: entry.Entry{
				Record: "Mon Jan 02 15:04:05 MST 2006",
			},
			expectErr: false,
			result: func() time.Time {
				t, _ := time.Parse(time.UnixDate, "Mon Jan 02 15:04:05 MST 2006")
				return t
			}(),
		},
		{
			name: "strptime-unix-field",
			cfg:  testConfigWithParseFrom("strptime", "%a %b %e %H:%M:%S %Z %Y", entry.NewField("some_field")),
			entry: entry.Entry{
				Record: map[string]interface{}{
					"some_field": "Mon Jan 2 15:04:05 MST 2006",
				},
			},
			expectErr: false,
			result: func() time.Time {
				t, _ := time.Parse(time.UnixDate, "Mon Jan 2 15:04:05 MST 2006")
				return t
			}(),
		},
		{
			name: "strptime-kitchen",
			cfg:  testConfig("strptime", "%H:%M%p"),
			entry: entry.Entry{
				Record: "12:34PM",
			},
			expectErr: false,
			result: func() time.Time {
				t, _ := time.Parse(time.Kitchen, "12:34PM")
				return t
			}(),
		},
		{
			name: "strptime-countdown",
			cfg:  testConfig("strptime", "%z %y %S %M %H %e %m"),
			entry: entry.Entry{
				Record: "-0100 01 01 01 01 01 01",
			},
			expectErr: false,
			result: func() time.Time {
				t, _ := time.Parse("-0700 06 05 04 03 02 01", "-0100 01 01 01 01 01 01")
				return t
			}(),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			buildContext := testutil.NewTestBuildContext(t)
			gotimePlugin, err := tc.cfg.Build(buildContext)
			require.NoError(t, err)

			mockOutput := &testutil.Plugin{}
			resultChan := make(chan *entry.Entry, 1)
			mockOutput.On("Process", mock.Anything, mock.Anything).Run(func(args mock.Arguments) {
				resultChan <- args.Get(1).(*entry.Entry)
			}).Return(nil)

			timeParser := gotimePlugin.(*TimeParser)
			timeParser.Output = mockOutput

			err = timeParser.Process(context.Background(), &tc.entry)

			if tc.expectErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}

			select {
			case e := <-resultChan:
				require.Equal(t, tc.result, e.Timestamp)
			case <-time.After(time.Second):
				require.FailNow(t, "Timed out waiting for entry to be processed")
			}
		})
	}

}
