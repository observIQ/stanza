package parser

import (
	"context"
	"testing"
	"time"

	"github.com/bluemedora/bplogagent/entry"
	"github.com/bluemedora/bplogagent/plugin"
	"github.com/bluemedora/bplogagent/plugin/helper"
	"github.com/bluemedora/bplogagent/plugin/mocks"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

type severityTestCase struct {
	name     string
	sample   interface{}
	mapping  map[interface{}]interface{}
	buildErr bool
	parseErr bool
	expected helper.Severity
}

func TestSeverityParser(t *testing.T) {

	testCases := []severityTestCase{
		{
			name:     "unknown",
			sample:   "blah",
			mapping:  nil,
			expected: helper.Default,
		},
		{
			name:     "error",
			sample:   "error",
			mapping:  nil,
			expected: helper.Error,
		},
		{
			name:     "error-capitalized",
			sample:   "Error",
			mapping:  nil,
			expected: helper.Error,
		},
		{
			name:     "error-all-caps",
			sample:   "ERROR",
			mapping:  nil,
			expected: helper.Error,
		},
		{
			name:     "custom-string",
			sample:   "NOOOOOOO",
			mapping:  map[interface{}]interface{}{"error": "NOOOOOOO"},
			expected: helper.Error,
		},
		{
			name:     "custom-string-caps-key",
			sample:   "NOOOOOOO",
			mapping:  map[interface{}]interface{}{"ErRoR": "NOOOOOOO"},
			expected: helper.Error,
		},
		{
			name:     "custom-int",
			sample:   1234,
			mapping:  map[interface{}]interface{}{"error": 1234},
			expected: helper.Error,
		},
		{
			name:     "mixed-list-string",
			sample:   "ThiS Is BaD",
			mapping:  map[interface{}]interface{}{"error": []interface{}{"NOOOOOOO", "this is bad", 1234}},
			expected: helper.Error,
		},
		{
			name:     "mixed-list-int",
			sample:   1234,
			mapping:  map[interface{}]interface{}{"error": []interface{}{"NOOOOOOO", "this is bad", 1234}},
			expected: helper.Error,
		},
		{
			name:     "overload-int-key",
			sample:   "E",
			mapping:  map[interface{}]interface{}{60: "E"},
			expected: helper.Error, // 60
		},
		{
			name:     "overload-native",
			sample:   "E",
			mapping:  map[interface{}]interface{}{helper.Error: "E"},
			expected: helper.Error, // 60
		},
		{
			name:     "custom-level",
			sample:   "weird",
			mapping:  map[interface{}]interface{}{12: "weird"},
			expected: 12,
		},
		{
			name:     "custom-level-list",
			sample:   "hey!",
			mapping:  map[interface{}]interface{}{16: []interface{}{"hey!", 1234}},
			expected: 16,
		},
		{
			name:     "custom-level-list-unfound",
			sample:   "not-in-the-list-but-thats-ok",
			mapping:  map[interface{}]interface{}{16: []interface{}{"hey!", 1234}},
			expected: helper.Default,
		},
		{
			name:     "custom-level-unbuildable",
			sample:   "not-in-the-list-but-thats-ok",
			mapping:  map[interface{}]interface{}{16: []interface{}{"hey!", 1234, 12.34}},
			buildErr: true,
		},
		{
			name:     "custom-level-list-unparseable",
			sample:   12.34,
			mapping:  map[interface{}]interface{}{16: []interface{}{"hey!", 1234}},
			parseErr: true,
		},
		{
			name:     "in-range",
			sample:   123,
			mapping:  map[interface{}]interface{}{"error": map[interface{}]interface{}{"min": 120, "max": 125}},
			expected: helper.Error,
		},
		{
			name:     "out-of-range",
			sample:   127,
			mapping:  map[interface{}]interface{}{"error": map[interface{}]interface{}{"min": 120, "max": 125}},
			expected: helper.Default,
		},
		{
			name:     "range-out-of-order",
			sample:   123,
			mapping:  map[interface{}]interface{}{"error": map[interface{}]interface{}{"min": 125, "max": 120}},
			expected: helper.Error,
		},
		{
			name:     "Http2xx-hit",
			sample:   201,
			mapping:  map[interface{}]interface{}{"error": "2xx"},
			expected: helper.Error,
		},
		{
			name:     "Http2xx-miss",
			sample:   301,
			mapping:  map[interface{}]interface{}{"error": "2xx"},
			expected: helper.Default,
		},
		{
			name:     "Http3xx-hit",
			sample:   301,
			mapping:  map[interface{}]interface{}{"error": "3xx"},
			expected: helper.Error,
		},
		{
			name:     "Http4xx-hit",
			sample:   "404",
			mapping:  map[interface{}]interface{}{"error": "4xx"},
			expected: helper.Error,
		},
		{
			name:     "Http5xx-hit",
			sample:   555,
			mapping:  map[interface{}]interface{}{"error": "5xx"},
			expected: helper.Error,
		},
		{
			name:     "Http-All",
			sample:   "301",
			mapping:  map[interface{}]interface{}{20: "2xx", 30: "3xx", 40: "4xx", 50: "5xx"},
			expected: 30,
		},
		{
			name:   "all-the-things-midrange",
			sample: 1234,
			mapping: map[interface{}]interface{}{
				"30":         "3xx",
				helper.Error: "4xx",
				"critical":   "5xx",
				helper.Trace: []interface{}{
					"ttttttracer",
					[]byte{100, 100, 100},
					map[interface{}]interface{}{"min": 1111, "max": 1234},
				},
				77: "",
			},
			expected: helper.Trace,
		},
		{
			name:   "all-the-things-bytes",
			sample: []byte{100, 100, 100},
			mapping: map[interface{}]interface{}{
				"30":         "3xx",
				helper.Error: "4xx",
				"critical":   "5xx",
				helper.Trace: []interface{}{
					"ttttttracer",
					[]byte{100, 100, 100},
					map[interface{}]interface{}{"min": 1111, "max": 1234},
				},
				77: "",
			},
			expected: helper.Trace,
		},
		{
			name:   "all-the-things-empty",
			sample: "",
			mapping: map[interface{}]interface{}{
				"30":         "3xx",
				helper.Error: "4xx",
				"critical":   "5xx",
				helper.Trace: []interface{}{
					"ttttttracer",
					[]byte{100, 100, 100},
					map[interface{}]interface{}{"min": 1111, "max": 1234},
				},
				77: "",
			},
			expected: 77,
		},
		{
			name:   "all-the-things-3xx",
			sample: "399",
			mapping: map[interface{}]interface{}{
				"30":         "3xx",
				helper.Error: "4xx",
				"critical":   "5xx",
				helper.Trace: []interface{}{
					"ttttttracer",
					[]byte{100, 100, 100},
					map[interface{}]interface{}{"min": 1111, "max": 1234},
				},
				77: "",
			},
			expected: 30,
		},
		{
			name:   "all-the-things-miss",
			sample: "miss",
			mapping: map[interface{}]interface{}{
				"30":         "3xx",
				helper.Error: "4xx",
				"critical":   "5xx",
				helper.Trace: []interface{}{
					"ttttttracer",
					[]byte{100, 100, 100},
					map[interface{}]interface{}{"min": 1111, "max": 2000},
				},
				77: "",
			},
			expected: helper.Default,
		},
	}

	rootField := entry.NewField()
	someField := entry.NewField("some_field")

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			rootCfg := parseSeverityTestConfig(rootField, tc.mapping)
			rootEntry := makeTestEntry(rootField, tc.sample)
			t.Run("root", runSeverityParseTest(t, rootCfg, rootEntry, tc.buildErr, tc.parseErr, tc.expected))

			nonRootCfg := parseSeverityTestConfig(someField, tc.mapping)
			nonRootEntry := makeTestEntry(someField, tc.sample)
			t.Run("non-root", runSeverityParseTest(t, nonRootCfg, nonRootEntry, tc.buildErr, tc.parseErr, tc.expected))
		})
	}
}

func runSeverityParseTest(t *testing.T, cfg *SeverityParserConfig, ent *entry.Entry, buildErr bool, parseErr bool, expected helper.Severity) func(*testing.T) {

	return func(t *testing.T) {
		buildContext := plugin.NewTestBuildContext(t)

		severityPlugin, err := cfg.Build(buildContext)
		if buildErr {
			require.Error(t, err, "expected error when configuring plugin")
			return
		}
		require.NoError(t, err, "unexpected error when configuring plugin")

		mockOutput := &mocks.Plugin{}
		resultChan := make(chan *entry.Entry, 1)
		mockOutput.On("Process", mock.Anything, mock.Anything).Run(func(args mock.Arguments) {
			resultChan <- args.Get(1).(*entry.Entry)
		}).Return(nil)

		severityParser := severityPlugin.(*SeverityParserPlugin)
		severityParser.Output = mockOutput

		err = severityParser.Process(context.Background(), ent)
		if parseErr {
			require.Error(t, err, "expected error when parsing sample")
			return
		}

		select {
		case e := <-resultChan:
			require.Equal(t, int(expected), e.Severity)
		case <-time.After(time.Second):
			require.FailNow(t, "Timed out waiting for entry to be processed")
		}
	}
}

func parseSeverityTestConfig(parseFrom entry.Field, mapping map[interface{}]interface{}) *SeverityParserConfig {
	return &SeverityParserConfig{
		TransformerConfig: helper.TransformerConfig{
			BasicConfig: helper.BasicConfig{
				PluginID:   "test_plugin_id",
				PluginType: "severity_parser",
			},
			OutputID: "output1",
		},
		SeverityParserConfig: helper.SeverityParserConfig{
			ParseFrom: parseFrom,
			Mapping:   mapping,
		},
	}
}
