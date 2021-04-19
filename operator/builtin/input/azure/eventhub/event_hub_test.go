package eventhub

import (
	"testing"

	"github.com/observiq/stanza/testutil"
	"github.com/stretchr/testify/require"
)

func TestBuild(t *testing.T) {
	cases := []struct {
		name      string
		input     EventHubInputConfig
		expectErr bool
	}{
		{
			"default-required",
			EventHubInputConfig{
				Namespace:        "test",
				Name:             "test",
				Group:            "test",
				ConnectionString: "test",
			},
			false,
		},
		{
			"default-required-prefetch",
			EventHubInputConfig{
				Namespace:        "test",
				Name:             "test",
				Group:            "test",
				ConnectionString: "test",
				PrefetchCount:    100,
			},
			false,
		},
		{
			"default-required-startat-end",
			EventHubInputConfig{
				Namespace:        "test",
				Name:             "test",
				Group:            "test",
				ConnectionString: "test",
				StartAt:          "end",
			},
			false,
		},
		{
			"default-required-startat-beginning",
			EventHubInputConfig{
				Namespace:        "test",
				Name:             "test",
				Group:            "test",
				ConnectionString: "test",
				StartAt:          "beginning",
			},
			false,
		},
		{
			"default-required-startat-invliad",
			EventHubInputConfig{
				Namespace:        "test",
				Name:             "test",
				Group:            "test",
				ConnectionString: "test",
				StartAt:          "invalid",
			},
			true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			cfg := NewEventHubConfig("test_id")
			cfg.Namespace = tc.input.Namespace
			cfg.Name = tc.input.Name
			cfg.Group = tc.input.Group
			cfg.ConnectionString = tc.input.ConnectionString

			if tc.input.PrefetchCount > 0 {
				cfg.PrefetchCount = tc.input.PrefetchCount
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
