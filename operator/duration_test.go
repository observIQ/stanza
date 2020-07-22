package operator

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	yaml "gopkg.in/yaml.v2"
)

func TestParseDuration(t *testing.T) {
	cases := []struct {
		name     string
		input    string
		expected Duration
	}{
		{
			"simple second",
			`"1s"`,
			Duration{time.Second},
		},
		{
			"simple minute",
			`"10m"`,
			Duration{10 * time.Minute},
		},
		{
			"number defaults to seconds",
			`10`,
			Duration{10 * time.Second},
		},
	}

	for _, tc := range cases {
		t.Run("yaml "+tc.name, func(t *testing.T) {
			var dur Duration
			err := yaml.UnmarshalStrict([]byte(tc.input), &dur)
			require.NoError(t, err)
			require.Equal(t, tc.expected, dur)
		})

		t.Run("json "+tc.name, func(t *testing.T) {
			var dur Duration
			err := json.Unmarshal([]byte(tc.input), &dur)
			require.NoError(t, err)
			require.Equal(t, tc.expected, dur)
		})
	}
}

func TestParseDurationRoundtrip(t *testing.T) {
	cases := []struct {
		name  string
		input Duration
	}{
		{
			"zero",
			Duration{},
		},
		{
			"second",
			Duration{time.Second},
		},
		{
			"minute",
			Duration{10 * time.Minute},
		},
	}

	for _, tc := range cases {
		t.Run("yaml "+tc.name, func(t *testing.T) {
			durBytes, err := yaml.Marshal(tc.input)
			require.NoError(t, err)

			var dur Duration
			err = yaml.UnmarshalStrict(durBytes, &dur)
			require.NoError(t, err)
			require.Equal(t, tc.input, dur)
		})

		t.Run("json "+tc.name, func(t *testing.T) {
			durBytes, err := json.Marshal(tc.input)
			require.NoError(t, err)

			var dur Duration
			err = json.Unmarshal(durBytes, &dur)
			require.NoError(t, err)
			require.Equal(t, tc.input, dur)
		})
	}
}
