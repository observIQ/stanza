package helper

import (
	"testing"

	"golang.org/x/text/encoding/unicode"

	"github.com/observiq/stanza/v2/testutil"
	otelhelper "github.com/open-telemetry/opentelemetry-log-collection/operator/helper"
	"github.com/stretchr/testify/require"
	"golang.org/x/text/encoding/japanese"
)

func TestNewEncodingConfig(t *testing.T) {
	expected := StanzaEncodingConfig{
		EncodingConfig: otelhelper.NewEncodingConfig(),
	}

	actual := NewEncodingConfig()
	require.Equal(t, expected, actual)
}

func TestStanzaEncoderBuild(t *testing.T) {
	testCases := []struct {
		desc        string
		encoding    string
		expected    otelhelper.Encoding
		expectError bool
	}{
		{
			desc:     "shift-jis",
			encoding: "shift-jis",
			expected: otelhelper.Encoding{
				Encoding: japanese.ShiftJIS,
			},
			expectError: false,
		},
		{
			desc:     "Supported by otel encoding config",
			encoding: "utf8",
			expected: otelhelper.Encoding{
				Encoding: unicode.UTF8,
			},
			expectError: false,
		},
		{
			desc:        "Not supported",
			encoding:    "bad_stuff",
			expected:    otelhelper.Encoding{},
			expectError: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			ec := NewEncodingConfig()
			ec.Encoding = tc.encoding

			actual, err := ec.Build(testutil.NewBuildContext(t))
			if tc.expectError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}

			require.Equal(t, tc.expected, actual)
		})
	}
}
