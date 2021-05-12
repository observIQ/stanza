package cloudwatch

import (
	"testing"
	"time"

	"github.com/observiq/stanza/operator/helper"
	"github.com/observiq/stanza/testutil"
	"github.com/stretchr/testify/require"
)

func TestBuild(t *testing.T) {
	test := "test"
	var testStreams = []*string{&test}
	cases := []struct {
		name      string
		input     CloudwatchInputConfig
		expectErr bool
	}{
		{
			"default",
			CloudwatchInputConfig{
				LogGroupName: "test",
				Region:       "test",
			},
			false,
		},
		{
			"log-stream-name-prefix",
			CloudwatchInputConfig{
				LogGroupName:        "test",
				LogStreamNamePrefix: "test",
				Region:              "test",
			},
			false,
		},
		{
			"event-limit",
			CloudwatchInputConfig{
				LogGroupName: "test",
				Region:       "test",
				EventLimit:   5000,
			},
			false,
		},
		{
			"poll-interval",
			CloudwatchInputConfig{
				LogGroupName: "test",
				Region:       "test",
				PollInterval: helper.NewDuration(time.Second * 10),
			},
			false,
		},
		{
			"profile",
			CloudwatchInputConfig{
				LogGroupName: "test",
				Region:       "test",
				Profile:      "test",
			},
			false,
		},
		{
			"log-stream-names",
			CloudwatchInputConfig{
				LogGroupName:   "test",
				LogStreamNames: testStreams,
				Region:         "test",
			},
			false,
		},
		{
			"startat-end",
			CloudwatchInputConfig{
				LogGroupName: "test",
				Region:       "test",
				StartAt:      "end",
			},
			false,
		},
		{
			"logStreamNames and logStreamNamePrefix both parameters Error",
			CloudwatchInputConfig{
				LogGroupName:        "test",
				LogStreamNames:      testStreams,
				LogStreamNamePrefix: "test",
				Region:              "test",
				StartAt:             "beginning",
			},
			true,
		},
		{
			"startat-beginning",
			CloudwatchInputConfig{
				LogGroupName: "test",
				Region:       "test",
			},
			false,
		},
		{
			"poll-interval-invalid",
			CloudwatchInputConfig{
				LogGroupName: "test",
				Region:       "test",
				PollInterval: helper.Duration{Duration: time.Second * 1},
			},
			true,
		},
		{
			"event-limit-invalid",
			CloudwatchInputConfig{
				LogGroupName: "test",
				Region:       "test",
				EventLimit:   10001,
			},
			true,
		},
		{
			"default-required-startat-invalid",
			CloudwatchInputConfig{
				LogGroupName: "test",
				Region:       "test",
				Profile:      "test",
				StartAt:      "invalid",
			},
			true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			cfg := NewCloudwatchConfig("test_id")
			cfg.LogGroupName = tc.input.LogGroupName
			cfg.Region = tc.input.Region
			cfg.Profile = tc.input.Profile

			if tc.input.LogStreamNamePrefix != "" {
				cfg.LogStreamNamePrefix = tc.input.LogStreamNamePrefix
			}

			if len(tc.input.LogStreamNames) > 0 {
				cfg.LogStreamNames = tc.input.LogStreamNames
			}

			if tc.input.EventLimit > 0 {
				cfg.EventLimit = tc.input.EventLimit
			}

			if tc.input.PollInterval.Raw() > time.Second*0 {
				cfg.PollInterval = tc.input.PollInterval
			}

			if tc.input.StartAt != "" {
				cfg.StartAt = tc.input.StartAt
			}

			_, err := cfg.Build(testutil.NewBuildContext(t))
			if tc.expectErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
		})
	}
}

func TestCurrentTimeInUnixMilliseconds(t *testing.T) {
	timeNow := time.Now()
	timeNowUnixMillis := timeNow.UnixNano() / int64(time.Millisecond)
	cases := []struct {
		name     string
		input    time.Time
		expected int64
	}{
		{
			name:     "test",
			input:    timeNow,
			expected: timeNowUnixMillis,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			x := currentTimeInUnixMilliseconds(tc.input)
			require.Equal(t, tc.expected, x)
		})
	}
}

func TestFromUnixMilli(t *testing.T) {
	timeNow := time.Now()
	timeNowUnixMillis := currentTimeInUnixMilliseconds(timeNow)

	cases := []struct {
		name     string
		input    int64
		expected time.Time
	}{
		{
			name:     "Time Now",
			input:    timeNowUnixMillis,
			expected: timeNow,
		},
		{
			name:     "Specific Time",
			input:    1620842185279,
			expected: time.Unix(0, 1620842185279000000),
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			x := fromUnixMilli(tc.input)
			require.Equal(t, tc.expected.Unix(), x.Unix())
		})
	}
}

func TestTimeLayoutParser(t *testing.T) {
	timeNow := time.Now().Unix()
	cases := []struct {
		name      string
		input     string
		timeToUse int64
		expected  string
	}{
		{
			name:      "Time Now",
			input:     "%Y/%m/%d",
			timeToUse: timeNow,
			expected:  time.Unix(timeNow, 0).Format("2006/01/02"),
		},
		{
			name:      "Year4Digigt-Month2Digit-Day2Digit",
			input:     "%Y-%m-%d",
			timeToUse: 1620843711,
			expected:  "2021-05-12",
		},
		{
			name:      "Year4Digigt-Month2Digit-Day2Digit-TrailingText",
			input:     "%Y-%m-%d/Test",
			timeToUse: 1620843711,
			expected:  "2021-05-12/Test",
		},
		{
			name:      "Layout repeated",
			input:     "%Y-%m-%d %Y-%m-%d",
			timeToUse: 1620843711,
			expected:  "2021-05-12 %Y-%m-%d",
		},
		{
			name:      "All Directives",
			input:     "%Y-%y-%m-%q-%b-%h-%B-%d-%g-%a-%A",
			timeToUse: 1639351311,
			expected:  "2021-21-12-12-Dec-Dec-December-12-12-Sun-Sunday",
		},
		{
			name:      "All Directives single digit day",
			input:     "%Y-%y-%m-%q-%b-%h-%B-%d-%g-%a-%A",
			timeToUse: 1619907711,
			expected:  "2021-21-05-5-May-May-May-01-1-Sat-Saturday",
		},
		{
			name:      "All Directives single digit month",
			input:     "%Y-%y-%m-%q-%b-%h-%B-%d-%g-%a-%A",
			timeToUse: 1620858111,
			expected:  "2021-21-05-5-May-May-May-12-12-Wed-Wednesday",
		},
		{
			name:      "Leap Year",
			input:     "%Y-%y-%m-%q-%b-%h-%B-%d-%g-%a-%A",
			timeToUse: 1583018511,
			expected:  "2020-20-02-2-Feb-Feb-February-29-29-Sat-Saturday",
		},
		{
			name:      "No Directives",
			input:     "2021-05-12",
			timeToUse: 1583018511,
			expected:  "2021-05-12",
		},
		{
			name:      "Empty string",
			input:     "",
			timeToUse: 1583018511,
			expected:  "",
		},
		{
			name:      "Symbols",
			input:     "%^&*!@#$()-=+_",
			timeToUse: 1583018511,
			expected:  "%^&*!@#$()-=+_",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			require.Equal(t, tc.expected, timeLayoutParser(tc.input, time.Unix(tc.timeToUse, 0)))
		})
	}
}
