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
