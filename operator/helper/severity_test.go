package helper

import (
	"fmt"
	"strings"
	"testing"

	"github.com/observiq/stanza/entry"
	"github.com/observiq/stanza/testutil"
	"github.com/stretchr/testify/require"
)

type severityTestCase struct {
	name       string
	sample     interface{}
	mappingSet string
	mapping    map[interface{}]interface{}
	buildErr   bool
	parseErr   bool
	expected   entry.Severity
}

func otlpSevCases() []severityTestCase {
	mustParse := map[string]entry.Severity{
		"tRaCe":  entry.Trace,
		"tRaCe2": entry.Trace2,
		"tRaCe3": entry.Trace3,
		"tRaCe4": entry.Trace4,
		"dEbUg":  entry.Debug,
		"dEbUg2": entry.Debug2,
		"dEbUg3": entry.Debug3,
		"dEbUg4": entry.Debug4,
		"iNFo":   entry.Info,
		"iNFo2":  entry.Info2,
		"iNFo3":  entry.Info3,
		"iNFo4":  entry.Info4,
		"wARn":   entry.Warning,
		"wARn2":  entry.Warning2,
		"wARn3":  entry.Warning3,
		"wARn4":  entry.Warning4,
		"eRrOr":  entry.Error,
		"eRrOr2": entry.Error2,
		"eRrOr3": entry.Error3,
		"eRrOr4": entry.Error4,
		"fAtAl":  entry.Emergency,
		"fAtAl2": entry.Emergency2,
		"fAtAl3": entry.Emergency3,
		"fAtAl4": entry.Emergency4,
	}

	cases := []severityTestCase{}
	for k, v := range mustParse {
		cases = append(cases,
			severityTestCase{
				name:     fmt.Sprintf("otlp-sev-%s-mIxEd", k),
				sample:   k,
				expected: v,
			},
			severityTestCase{
				name:     fmt.Sprintf("otlp-sev-%s-lower", k),
				sample:   strings.ToLower(k),
				expected: v,
			},
			severityTestCase{
				name:     fmt.Sprintf("otlp-sev-%s-upper", k),
				sample:   strings.ToUpper(k),
				expected: v,
			},
			severityTestCase{
				name:     fmt.Sprintf("otlp-sev-%s-title", k),
				sample:   strings.ToTitle(k),
				expected: v,
			})
	}
	return cases
}

func TestSeverityParser(t *testing.T) {

	testCases := []severityTestCase{
		{
			name:     "unknown",
			sample:   "blah",
			mapping:  nil,
			expected: entry.Default,
		},
		{
			name:     "error",
			sample:   "error",
			mapping:  nil,
			expected: entry.Error,
		},
		{
			name:     "error-capitalized",
			sample:   "Error",
			mapping:  nil,
			expected: entry.Error,
		},
		{
			name:     "error-all-caps",
			sample:   "ERROR",
			mapping:  nil,
			expected: entry.Error,
		},
		{
			name:     "custom-string",
			sample:   "NOOOOOOO",
			mapping:  map[interface{}]interface{}{"error": "NOOOOOOO"},
			expected: entry.Error,
		},
		{
			name:     "custom-string-caps-key",
			sample:   "NOOOOOOO",
			mapping:  map[interface{}]interface{}{"ErRoR": "NOOOOOOO"},
			expected: entry.Error,
		},
		{
			name:     "custom-int",
			sample:   1234,
			mapping:  map[interface{}]interface{}{"error": 1234},
			expected: entry.Error,
		},
		{
			name:     "custom-float64",
			sample:   float64(6),
			mapping:  map[interface{}]interface{}{"error": 6},
			expected: entry.Error,
		},
		{
			name:     "mixed-list-string",
			sample:   "ThiS Is BaD",
			mapping:  map[interface{}]interface{}{"error": []interface{}{"NOOOOOOO", "this is bad", 1234}},
			expected: entry.Error,
		},
		{
			name:     "mixed-list-int",
			sample:   1234,
			mapping:  map[interface{}]interface{}{"error": []interface{}{"NOOOOOOO", "this is bad", 1234}},
			expected: entry.Error,
		},
		{
			name:     "overload-int-key",
			sample:   "E",
			mapping:  map[interface{}]interface{}{60: "E"},
			expected: entry.Error, // 60
		},
		{
			name:     "overload-native",
			sample:   "E",
			mapping:  map[interface{}]interface{}{int(entry.Error): "E"},
			expected: entry.Error, // 60
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
			expected: entry.Default,
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
			expected: entry.Error,
		},
		{
			name:     "in-range-min",
			sample:   120,
			mapping:  map[interface{}]interface{}{"error": map[interface{}]interface{}{"min": 120, "max": 125}},
			expected: entry.Error,
		},
		{
			name:     "in-range-max",
			sample:   125,
			mapping:  map[interface{}]interface{}{"error": map[interface{}]interface{}{"min": 120, "max": 125}},
			expected: entry.Error,
		},
		{
			name:     "out-of-range-min-minus",
			sample:   119,
			mapping:  map[interface{}]interface{}{"error": map[interface{}]interface{}{"min": 120, "max": 125}},
			expected: entry.Default,
		},
		{
			name:     "out-of-range-max-plus",
			sample:   126,
			mapping:  map[interface{}]interface{}{"error": map[interface{}]interface{}{"min": 120, "max": 125}},
			expected: entry.Default,
		},
		{
			name:     "range-out-of-order",
			sample:   123,
			mapping:  map[interface{}]interface{}{"error": map[interface{}]interface{}{"min": 125, "max": 120}},
			expected: entry.Error,
		},
		{
			name:     "Http2xx-hit",
			sample:   201,
			mapping:  map[interface{}]interface{}{"error": "2xx"},
			expected: entry.Error,
		},
		{
			name:     "Http2xx-miss",
			sample:   301,
			mapping:  map[interface{}]interface{}{"error": "2xx"},
			expected: entry.Default,
		},
		{
			name:     "Http3xx-hit",
			sample:   301,
			mapping:  map[interface{}]interface{}{"error": "3xx"},
			expected: entry.Error,
		},
		{
			name:     "Http4xx-hit",
			sample:   "404",
			mapping:  map[interface{}]interface{}{"error": "4xx"},
			expected: entry.Error,
		},
		{
			name:     "Http5xx-hit",
			sample:   555,
			mapping:  map[interface{}]interface{}{"error": "5xx"},
			expected: entry.Error,
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
				"30":             "3xx",
				int(entry.Error): "4xx",
				"critical":       "5xx",
				int(entry.Trace): []interface{}{
					"ttttttracer",
					[]byte{100, 100, 100},
					map[interface{}]interface{}{"min": 1111, "max": 1234},
				},
				77: "",
			},
			expected: entry.Trace,
		},
		{
			name:   "all-the-things-bytes",
			sample: []byte{100, 100, 100},
			mapping: map[interface{}]interface{}{
				"30":             "3xx",
				int(entry.Error): "4xx",
				"critical":       "5xx",
				int(entry.Trace): []interface{}{
					"ttttttracer",
					[]byte{100, 100, 100},
					map[interface{}]interface{}{"min": 1111, "max": 1234},
				},
				77: "",
			},
			expected: entry.Trace,
		},
		{
			name:   "all-the-things-empty",
			sample: "",
			mapping: map[interface{}]interface{}{
				"30":             "3xx",
				int(entry.Error): "4xx",
				"critical":       "5xx",
				int(entry.Trace): []interface{}{
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
				"30":             "3xx",
				int(entry.Error): "4xx",
				"critical":       "5xx",
				int(entry.Trace): []interface{}{
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
				"30":             "3xx",
				int(entry.Error): "4xx",
				"critical":       "5xx",
				int(entry.Trace): []interface{}{
					"ttttttracer",
					[]byte{100, 100, 100},
					map[interface{}]interface{}{"min": 1111, "max": 2000},
				},
				77: "",
			},
			expected: entry.Default,
		},
		{
			name:       "base-mapping-none",
			sample:     "error",
			mappingSet: "none",
			mapping:    nil,
			expected:   entry.Default, // not error
		},
	}

	testCases = append(testCases, otlpSevCases()...)

	rootField := entry.NewRecordField()
	someField := entry.NewRecordField("some_field")

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Run("root", tc.run(rootField))
			t.Run("non-root", tc.run(someField))
		})
	}
}

func (tc severityTestCase) run(parseFrom entry.Field) func(*testing.T) {

	return func(t *testing.T) {
		t.Parallel()

		buildContext := testutil.NewBuildContext(t)

		cfg := &SeverityParserConfig{
			ParseFrom: &parseFrom,
			Preset:    tc.mappingSet,
			Mapping:   tc.mapping,
		}

		severityParser, err := cfg.Build(buildContext)
		if tc.buildErr {
			require.Error(t, err, "expected error when configuring operator")
			return
		}
		require.NoError(t, err, "unexpected error when configuring operator")

		ent := entry.New()
		ent.Set(parseFrom, tc.sample)
		err = severityParser.Parse(ent)
		if tc.parseErr {
			require.Error(t, err, "expected error when parsing sample")
			return
		}
		require.NoError(t, err)

		require.Equal(t, tc.expected, ent.Severity)

	}
}
