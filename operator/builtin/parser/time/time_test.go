package time

import (
	"context"
	"fmt"
	"math"
	"strconv"
	"testing"
	"time"

	"github.com/observiq/stanza/entry"
	"github.com/observiq/stanza/operator"
	"github.com/observiq/stanza/operator/helper"
	"github.com/observiq/stanza/testutil"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestIsZero(t *testing.T) {
	require.True(t, (&helper.TimeParser{}).IsZero())
	require.False(t, (&helper.TimeParser{Layout: "strptime"}).IsZero())
}

func TestTimeUnixParser(t *testing.T) {
	tests := []struct {
		name       string
		layoutType string
		layout     string
		sample     interface{}
		want       string
	}{
		{
			name:       "unixhex now string",
			layoutType: helper.UnixHex,
			layout:     ``,
			sample:     strconv.FormatInt(time.Now().Unix(), 16),
			want:       ``,
		},
		{
			name:       "unixhex now byte slice",
			layoutType: helper.UnixHex,
			layout:     ``,
			sample:     []byte(strconv.FormatInt(time.Now().Unix(), 16)),
			want:       ``,
		},
	}

	rootField := entry.NewRecordField()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var hexStamp string
			switch v := tt.sample.(type) {
			case []byte:
				hexStamp = string(v)
			case string:
				hexStamp = fmt.Sprintf("%v", v)
			default:
				return
			}

			expected, err := strconv.ParseInt(hexStamp, 16, 64)
			require.NoError(t, err)
			gotimeRootCfg := parseTimeTestConfig(tt.layoutType, tt.layout, rootField)
			t.Run(tt.name+"-root", runTimeParseTest(t, gotimeRootCfg, makeTestEntry(rootField, tt.sample), false, false, time.Unix(expected, 0)))

		})
	}
}

func TestTimeParser(t *testing.T) {

	testCases := []struct {
		name           string
		sample         interface{}
		gotimeLayout   string
		strptimeLayout string
	}{
		{
			name:           "unix",
			sample:         "Mon Jan 2 15:04:05 MST 2006",
			gotimeLayout:   "Mon Jan 2 15:04:05 MST 2006",
			strptimeLayout: "%a %b %e %H:%M:%S %Z %Y",
		},
		{
			name:           "almost-unix",
			sample:         "Mon Jan 02 15:04:05 MST 2006",
			gotimeLayout:   "Mon Jan 02 15:04:05 MST 2006",
			strptimeLayout: "%a %b %d %H:%M:%S %Z %Y",
		},
		{
			name:           "kitchen",
			sample:         "12:34PM",
			gotimeLayout:   time.Kitchen,
			strptimeLayout: "%H:%M%p",
		},
		{
			name:           "kitchen-bytes",
			sample:         []byte("12:34PM"),
			gotimeLayout:   time.Kitchen,
			strptimeLayout: "%H:%M%p",
		},
		{
			name:           "countdown",
			sample:         "-0100 01 01 01 01 01 01",
			gotimeLayout:   "-0700 06 05 04 03 02 01",
			strptimeLayout: "%z %y %S %M %H %e %m",
		},
		{
			name:           "debian-syslog",
			sample:         "Jun 09 11:39:45",
			gotimeLayout:   "Jan 02 15:04:05",
			strptimeLayout: "%b %d %H:%M:%S",
		},
		{
			name:           "opendistro",
			sample:         "2020-06-09T15:39:58",
			gotimeLayout:   "2006-01-02T15:04:05",
			strptimeLayout: "%Y-%m-%dT%H:%M:%S",
		},
		{
			name:           "postgres",
			sample:         "2019-11-05 10:38:35.118 EST",
			gotimeLayout:   "2006-01-02 15:04:05.999 MST",
			strptimeLayout: "%Y-%m-%d %H:%M:%S.%L %Z",
		},
		{
			name:           "ibm-mq",
			sample:         "3/4/2018 11:52:29",
			gotimeLayout:   "1/2/2006 15:04:05",
			strptimeLayout: "%q/%g/%Y %H:%M:%S",
		},
		{
			name:           "cassandra",
			sample:         "2019-11-27T09:34:32.901-0500",
			gotimeLayout:   "2006-01-02T15:04:05.999-0700",
			strptimeLayout: "%Y-%m-%dT%H:%M:%S.%L%z",
		},
		{
			name:           "oracle",
			sample:         "2019-10-15T10:42:01.900436-04:00",
			gotimeLayout:   "2006-01-02T15:04:05.999999-07:00",
			strptimeLayout: "%Y-%m-%dT%H:%M:%S.%f%j",
		},
		{
			name:           "oracle-listener",
			sample:         "22-JUL-2019 15:16:13",
			gotimeLayout:   "02-Jan-2006 15:04:05",
			strptimeLayout: "%d-%b-%Y %H:%M:%S",
		},
		{
			name:           "k8s",
			sample:         "2019-03-08T18:41:12.152531115Z",
			gotimeLayout:   "2006-01-02T15:04:05.999999999Z",
			strptimeLayout: "%Y-%m-%dT%H:%M:%S.%sZ",
		},
		{
			name:           "jetty",
			sample:         "05/Aug/2019:20:38:46 +0000",
			gotimeLayout:   "02/Jan/2006:15:04:05 -0700",
			strptimeLayout: "%d/%b/%Y:%H:%M:%S %z",
		},
		{
			name:           "puppet",
			sample:         "Aug  4 03:26:02",
			gotimeLayout:   "Jan _2 15:04:05",
			strptimeLayout: "%b %e %H:%M:%S",
		},
	}

	rootField := entry.NewRecordField()
	someField := entry.NewRecordField("some_field")

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			var sampleStr string
			switch s := tc.sample.(type) {
			case string:
				sampleStr = s
			case []byte:
				sampleStr = string(s)
			default:
				require.FailNow(t, "unexpected sample type")
			}

			expected, err := time.ParseInLocation(tc.gotimeLayout, sampleStr, time.Local)
			require.NoError(t, err, "Test configuration includes invalid timestamp or layout")

			gotimeRootCfg := parseTimeTestConfig(helper.GotimeKey, tc.gotimeLayout, rootField)
			t.Run("gotime-root", runTimeParseTest(t, gotimeRootCfg, makeTestEntry(rootField, tc.sample), false, false, expected))

			gotimeNonRootCfg := parseTimeTestConfig(helper.GotimeKey, tc.gotimeLayout, someField)
			t.Run("gotime-non-root", runTimeParseTest(t, gotimeNonRootCfg, makeTestEntry(someField, tc.sample), false, false, expected))

			strptimeRootCfg := parseTimeTestConfig(helper.StrptimeKey, tc.strptimeLayout, rootField)
			t.Run("strptime-root", runTimeParseTest(t, strptimeRootCfg, makeTestEntry(rootField, tc.sample), false, false, expected))

			strptimeNonRootCfg := parseTimeTestConfig(helper.StrptimeKey, tc.strptimeLayout, someField)
			t.Run("strptime-non-root", runTimeParseTest(t, strptimeNonRootCfg, makeTestEntry(someField, tc.sample), false, false, expected))
		})
	}
}

func TestTimeEpochs(t *testing.T) {

	testCases := []struct {
		name     string
		sample   interface{}
		layout   string
		expected time.Time
		maxLoss  time.Duration
	}{
		{
			name:     "s-default-string",
			sample:   "1136214245",
			layout:   "s",
			expected: time.Unix(1136214245, 0),
		},
		{
			name:     "s-default-bytes",
			sample:   []byte("1136214245"),
			layout:   "s",
			expected: time.Unix(1136214245, 0),
		},
		{
			name:     "s-default-int",
			sample:   1136214245,
			layout:   "s",
			expected: time.Unix(1136214245, 0),
		},
		{
			name:     "s-default-float",
			sample:   1136214245.0,
			layout:   "s",
			expected: time.Unix(1136214245, 0),
		},
		{
			name:     "ms-default-string",
			sample:   "1136214245123",
			layout:   "ms",
			expected: time.Unix(1136214245, 123000000),
		},
		{
			name:     "ms-default-int",
			sample:   1136214245123,
			layout:   "ms",
			expected: time.Unix(1136214245, 123000000),
		},
		{
			name:     "ms-default-float",
			sample:   1136214245123.0,
			layout:   "ms",
			expected: time.Unix(1136214245, 123000000),
		},
		{
			name:     "us-default-string",
			sample:   "1136214245123456",
			layout:   "us",
			expected: time.Unix(1136214245, 123456000),
		},
		{
			name:     "us-default-int",
			sample:   1136214245123456,
			layout:   "us",
			expected: time.Unix(1136214245, 123456000),
		},
		{
			name:     "us-default-float",
			sample:   1136214245123456.0,
			layout:   "us",
			expected: time.Unix(1136214245, 123456000),
		},
		{
			name:     "ns-default-string",
			sample:   "1136214245123456789",
			layout:   "ns",
			expected: time.Unix(1136214245, 123456789),
		},
		{
			name:     "ns-default-int",
			sample:   1136214245123456789,
			layout:   "ns",
			expected: time.Unix(1136214245, 123456789),
		},
		{
			name:     "ns-default-float",
			sample:   1136214245123456789.0,
			layout:   "ns",
			expected: time.Unix(1136214245, 123456789),
			maxLoss:  time.Nanosecond * 100,
		},
		{
			name:     "s.ms-default-string",
			sample:   "1136214245.123",
			layout:   "s.ms",
			expected: time.Unix(1136214245, 123000000),
		},
		{
			name:     "s.ms-default-int",
			sample:   1136214245,
			layout:   "s.ms",
			expected: time.Unix(1136214245, 0), // drops subseconds
			maxLoss:  time.Nanosecond * 100,
		},
		{
			name:     "s.ms-default-float",
			sample:   1136214245.123,
			layout:   "s.ms",
			expected: time.Unix(1136214245, 123000000),
		},
		{
			name:     "s.us-default-string",
			sample:   "1136214245.123456",
			layout:   "s.us",
			expected: time.Unix(1136214245, 123456000),
		},
		{
			name:     "s.us-default-int",
			sample:   1136214245,
			layout:   "s.us",
			expected: time.Unix(1136214245, 0), // drops subseconds
			maxLoss:  time.Nanosecond * 100,
		},
		{
			name:     "s.us-default-float",
			sample:   1136214245.123456,
			layout:   "s.us",
			expected: time.Unix(1136214245, 123456000),
		},
		{
			name:     "s.ns-default-string",
			sample:   "1136214245.123456789",
			layout:   "s.ns",
			expected: time.Unix(1136214245, 123456789),
		},
		{
			name:     "s.ns-default-int",
			sample:   1136214245,
			layout:   "s.ns",
			expected: time.Unix(1136214245, 0), // drops subseconds
			maxLoss:  time.Nanosecond * 100,
		},
		{
			name:     "s.ns-default-float",
			sample:   1136214245.123456789,
			layout:   "s.ns",
			expected: time.Unix(1136214245, 123456789),
			maxLoss:  time.Nanosecond * 100,
		},
	}

	rootField := entry.NewRecordField()
	someField := entry.NewRecordField("some_field")

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			rootCfg := parseTimeTestConfig(helper.EpochKey, tc.layout, rootField)
			t.Run("epoch-root", runLossyTimeParseTest(t, rootCfg, makeTestEntry(rootField, tc.sample), false, false, tc.expected, tc.maxLoss))

			nonRootCfg := parseTimeTestConfig(helper.EpochKey, tc.layout, someField)
			t.Run("epoch-non-root", runLossyTimeParseTest(t, nonRootCfg, makeTestEntry(someField, tc.sample), false, false, tc.expected, tc.maxLoss))
		})
	}
}

func TestTimeErrors(t *testing.T) {

	testCases := []struct {
		name       string
		sample     interface{}
		layoutType string
		layout     string
		buildErr   bool
		parseErr   bool
	}{
		{
			name:       "bad-layout-type",
			layoutType: "fake",
			buildErr:   true,
		},
		{
			name:       "bad-strptime-directive",
			layoutType: "strptime",
			layout:     "%1",
			buildErr:   true,
		},
		{
			name:       "bad-epoch-layout",
			layoutType: "epoch",
			layout:     "years",
			buildErr:   true,
		},
		{
			name:       "bad-native-value",
			layoutType: "native",
			sample:     1,
			parseErr:   true,
		},
		{
			name:       "bad-gotime-value",
			layoutType: "gotime",
			layout:     time.Kitchen,
			sample:     1,
			parseErr:   true,
		},
		{
			name:       "bad-epoch-value",
			layoutType: "epoch",
			layout:     "s",
			sample:     "not-a-number",
			parseErr:   true,
		},
		{
			name:       "bad-unixhex-value",
			layoutType: helper.UnixHex,
			layout:     "",
			sample:     time.Now().Unix(),
			parseErr:   true,
		},
	}

	rootField := entry.NewRecordField()
	someField := entry.NewRecordField("some_field")

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			rootCfg := parseTimeTestConfig(tc.layoutType, tc.layout, rootField)
			t.Run("err-root", runTimeParseTest(t, rootCfg, makeTestEntry(rootField, tc.sample), tc.buildErr, tc.parseErr, time.Now()))

			nonRootCfg := parseTimeTestConfig(tc.layoutType, tc.layout, someField)
			t.Run("err-non-root", runTimeParseTest(t, nonRootCfg, makeTestEntry(someField, tc.sample), tc.buildErr, tc.parseErr, time.Now()))
		})
	}
}

func makeTestEntry(field entry.Field, value interface{}) *entry.Entry {
	e := entry.New()
	e.Set(field, value)
	return e
}

func runTimeParseTest(t *testing.T, cfg *TimeParserConfig, ent *entry.Entry, buildErr bool, parseErr bool, expected time.Time) func(*testing.T) {
	return runLossyTimeParseTest(t, cfg, ent, buildErr, parseErr, expected, time.Duration(0))
}

func runLossyTimeParseTest(t *testing.T, cfg *TimeParserConfig, ent *entry.Entry, buildErr bool, parseErr bool, expected time.Time, maxLoss time.Duration) func(*testing.T) {

	return func(t *testing.T) {
		buildContext := testutil.NewBuildContext(t)

		ops, err := cfg.Build(buildContext)
		if buildErr {
			require.Error(t, err, "expected error when configuring operator")
			return
		}
		require.NoError(t, err)
		op := ops[0]

		mockOutput := &testutil.Operator{}
		resultChan := make(chan *entry.Entry, 1)
		mockOutput.On("Process", mock.Anything, mock.Anything).Run(func(args mock.Arguments) {
			resultChan <- args.Get(1).(*entry.Entry)
		}).Return(nil)

		timeParser := op.(*TimeParserOperator)
		timeParser.OutputOperators = []operator.Operator{mockOutput}

		err = timeParser.Process(context.Background(), ent)
		if parseErr {
			require.Error(t, err, "expected error when configuring operator")
			return
		}
		require.NoError(t, err)

		select {
		case e := <-resultChan:
			diff := time.Duration(math.Abs(float64(expected.Sub(e.Timestamp))))
			require.True(t, diff <= maxLoss)
		case <-time.After(time.Second):
			require.FailNow(t, "Timed out waiting for entry to be processed")
		}
	}
}

func parseTimeTestConfig(layoutType, layout string, parseFrom entry.Field) *TimeParserConfig {
	cfg := NewTimeParserConfig("test_operator_id")
	cfg.OutputIDs = []string{"output1"}
	cfg.TimeParser = helper.TimeParser{
		LayoutType: layoutType,
		Layout:     layout,
		ParseFrom:  &parseFrom,
	}
	return cfg
}
