package otlp

import (
	"fmt"
	"testing"
	"time"

	"github.com/observiq/stanza/entry"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/consumer/pdata"
)

func BenchmarkConvertSimple(b *testing.B) {
	for i := 0; i < b.N; i++ {
		Convert([]*entry.Entry{entry.New()})
	}
}

func BenchmarkConvertComplex(b *testing.B) {
	for i := 0; i < b.N; i++ {
		b.StopTimer()
		e := complexEntry()
		b.StartTimer()
		Convert([]*entry.Entry{e})
	}
}

func complexEntry() *entry.Entry {
	e := entry.New()
	e.Severity = entry.Error
	e.AddResourceKey("type", "global")
	e.AddLabel("one", "two")
	e.AddLabel("two", "three")
	e.Record = map[string]interface{}{
		"bool":   true,
		"int":    123,
		"double": 12.34,
		"string": "hello",
		"bytes":  []byte("asdf"),
		"object": map[string]interface{}{
			"bool":   true,
			"int":    123,
			"double": 12.34,
			"string": "hello",
			"bytes":  []byte("asdf"),
			"object": map[string]interface{}{
				"bool":   true,
				"int":    123,
				"double": 12.34,
				"string": "hello",
				"bytes":  []byte("asdf"),
			},
		},
	}
	return e
}

func TestConvertMetadata(t *testing.T) {

	now := time.Now()

	e := entry.New()
	e.Timestamp = now
	e.Severity = entry.Error
	e.AddResourceKey("type", "global")
	e.AddLabel("one", "two")
	e.Record = true

	result := Convert([]*entry.Entry{e})

	resourceLogs := result.ResourceLogs()
	require.Equal(t, 1, resourceLogs.Len(), "expected 1 resource")

	libLogs := resourceLogs.At(0).InstrumentationLibraryLogs()
	require.Equal(t, 1, libLogs.Len(), "expected 1 library")

	logSlice := libLogs.At(0).Logs()
	require.Equal(t, 1, logSlice.Len(), "expected 1 log")

	log := logSlice.At(0)
	require.Equal(t, now.UnixNano(), int64(log.Timestamp()))

	require.Equal(t, pdata.SeverityNumberERROR, log.SeverityNumber())
	require.Equal(t, "", log.SeverityText())

	atts := log.Attributes()
	require.Equal(t, 1, atts.Len(), "expected 1 attribute")
	attVal, ok := atts.Get("one")
	require.True(t, ok, "expected label with key 'one'")
	require.Equal(t, "two", attVal.StringVal(), "expected label to have value 'two'")

	bod := log.Body()
	require.Equal(t, pdata.AttributeValueBOOL, bod.Type())
	require.True(t, bod.BoolVal())
}

func TestConvertSimpleBody(t *testing.T) {

	require.True(t, recordToBody(true).BoolVal())
	require.False(t, recordToBody(false).BoolVal())

	require.Equal(t, "string", recordToBody("string").StringVal())
	require.Equal(t, "bytes", recordToBody([]byte("bytes")).StringVal())

	require.Equal(t, int64(1), recordToBody(1).IntVal())
	require.Equal(t, int64(1), recordToBody(int8(1)).IntVal())
	require.Equal(t, int64(1), recordToBody(int16(1)).IntVal())
	require.Equal(t, int64(1), recordToBody(int32(1)).IntVal())
	require.Equal(t, int64(1), recordToBody(int64(1)).IntVal())

	require.Equal(t, int64(1), recordToBody(uint(1)).IntVal())
	require.Equal(t, int64(1), recordToBody(uint8(1)).IntVal())
	require.Equal(t, int64(1), recordToBody(uint16(1)).IntVal())
	require.Equal(t, int64(1), recordToBody(uint32(1)).IntVal())
	require.Equal(t, int64(1), recordToBody(uint64(1)).IntVal())

	require.Equal(t, float64(1), recordToBody(float32(1)).DoubleVal())
	require.Equal(t, float64(1), recordToBody(float64(1)).DoubleVal())
}

func TestConvertMapBody(t *testing.T) {
	structuredRecord := map[string]interface{}{
		"true":    true,
		"false":   false,
		"string":  "string",
		"bytes":   []byte("bytes"),
		"int":     1,
		"int8":    int8(1),
		"int16":   int16(1),
		"int32":   int32(1),
		"int64":   int64(1),
		"uint":    uint(1),
		"uint8":   uint8(1),
		"uint16":  uint16(1),
		"uint32":  uint32(1),
		"uint64":  uint64(1),
		"float32": float32(1),
		"float64": float64(1),
		"strings": []string{"foo", "bar"},
	}

	result := recordToBody(structuredRecord).MapVal()

	v, _ := result.Get("true")
	require.True(t, v.BoolVal())
	v, _ = result.Get("false")
	require.False(t, v.BoolVal())

	for _, k := range []string{"string", "bytes"} {
		v, _ = result.Get(k)
		require.Equal(t, k, v.StringVal())
	}
	for _, k := range []string{"int", "int8", "int16", "int32", "int64", "uint", "uint8", "uint16", "uint32", "uint64"} {
		v, _ = result.Get(k)
		require.Equal(t, int64(1), v.IntVal())
	}
	for _, k := range []string{"float32", "float64"} {
		v, _ = result.Get(k)
		require.Equal(t, float64(1), v.DoubleVal())
	}

	v, _ = result.Get("strings")
	require.Equal(t, 2, v.ArrayVal().Len())
	require.Equal(t, "foo", v.ArrayVal().At(0).StringVal())
	require.Equal(t, "bar", v.ArrayVal().At(1).StringVal())
}

func TestConvertArrayBody(t *testing.T) {
	structuredRecord := []interface{}{
		true,
		false,
		"string",
		[]byte("bytes"),
		1,
		int8(1),
		int16(1),
		int32(1),
		int64(1),
		uint(1),
		uint8(1),
		uint16(1),
		uint32(1),
		uint64(1),
		float32(1),
		float64(1),
		[]interface{}{"string", 1},
		map[string]interface{}{"one": 1, "yes": true},
		map[string]string{"foo": "bar"},
	}

	result := recordToBody(structuredRecord).ArrayVal()

	require.True(t, result.At(0).BoolVal())
	require.False(t, result.At(1).BoolVal())
	require.Equal(t, "string", result.At(2).StringVal())
	require.Equal(t, "bytes", result.At(3).StringVal())

	require.Equal(t, int64(1), result.At(4).IntVal())  // int
	require.Equal(t, int64(1), result.At(5).IntVal())  // int8
	require.Equal(t, int64(1), result.At(6).IntVal())  // int16
	require.Equal(t, int64(1), result.At(7).IntVal())  // int32
	require.Equal(t, int64(1), result.At(8).IntVal())  // int64
	require.Equal(t, int64(1), result.At(9).IntVal())  // uint
	require.Equal(t, int64(1), result.At(10).IntVal()) // uint8
	require.Equal(t, int64(1), result.At(11).IntVal()) // uint16
	require.Equal(t, int64(1), result.At(12).IntVal()) // uint32
	require.Equal(t, int64(1), result.At(13).IntVal()) // uint64

	require.Equal(t, float64(1), result.At(14).DoubleVal()) // float32
	require.Equal(t, float64(1), result.At(15).DoubleVal()) // float64

	nestedArr := result.At(16).ArrayVal()
	require.Equal(t, "string", nestedArr.At(0).StringVal())
	require.Equal(t, int64(1), nestedArr.At(1).IntVal())

	nestedMap := result.At(17).MapVal()
	v, _ := nestedMap.Get("one")
	require.Equal(t, int64(1), v.IntVal())
	v, _ = nestedMap.Get("yes")
	require.True(t, v.BoolVal())

	stringsMap := result.At(18).MapVal()
	v, _ = stringsMap.Get("foo")
	require.Equal(t, "bar", v.StringVal())
}

func TestConvertUnknownBody(t *testing.T) {
	unknownType := map[string]int{"0": 0, "1": 1}
	require.Equal(t, fmt.Sprintf("%v", unknownType), recordToBody(unknownType).StringVal())
}

func TestConvertNestedMapBody(t *testing.T) {

	unknownType := map[string]int{"0": 0, "1": 1}

	structuredRecord := map[string]interface{}{
		"array":   []interface{}{0, 1},
		"map":     map[string]interface{}{"0": 0, "1": "one"},
		"unknown": unknownType,
	}

	result := recordToBody(structuredRecord).MapVal()

	arrayAttVal, _ := result.Get("array")
	a := arrayAttVal.ArrayVal()
	require.Equal(t, int64(0), a.At(0).IntVal())
	require.Equal(t, int64(1), a.At(1).IntVal())

	mapAttVal, _ := result.Get("map")
	m := mapAttVal.MapVal()
	v, _ := m.Get("0")
	require.Equal(t, int64(0), v.IntVal())
	v, _ = m.Get("1")
	require.Equal(t, "one", v.StringVal())

	unknownAttVal, _ := result.Get("unknown")
	require.Equal(t, fmt.Sprintf("%v", unknownType), unknownAttVal.StringVal())
}

func recordToBody(record interface{}) pdata.AttributeValue {
	e := entry.New()
	e.Record = record
	return convertAndDrill(e).Body()
}

func convertAndDrill(e *entry.Entry) pdata.LogRecord {
	return Convert([]*entry.Entry{e}).ResourceLogs().At(0).InstrumentationLibraryLogs().At(0).Logs().At(0)
}

func TestConvertSeverity(t *testing.T) {
	cases := []struct {
		severity       entry.Severity
		expectedNumber pdata.SeverityNumber
	}{
		{entry.Default, pdata.SeverityNumberUNDEFINED},
		{5, pdata.SeverityNumberTRACE},
		{entry.Trace, pdata.SeverityNumberTRACE},
		{11, pdata.SeverityNumberTRACE},
		{entry.Trace2, pdata.SeverityNumberTRACE2},
		{entry.Trace3, pdata.SeverityNumberTRACE3},
		{entry.Trace4, pdata.SeverityNumberTRACE4},
		{15, pdata.SeverityNumberTRACE4},
		{entry.Debug, pdata.SeverityNumberDEBUG},
		{21, pdata.SeverityNumberDEBUG},
		{entry.Debug2, pdata.SeverityNumberDEBUG2},
		{entry.Debug3, pdata.SeverityNumberDEBUG3},
		{entry.Debug4, pdata.SeverityNumberDEBUG4},
		{25, pdata.SeverityNumberDEBUG4},
		{entry.Info, pdata.SeverityNumberINFO},
		{31, pdata.SeverityNumberINFO},
		{entry.Info2, pdata.SeverityNumberINFO2},
		{entry.Info3, pdata.SeverityNumberINFO3},
		{entry.Info4, pdata.SeverityNumberINFO4},
		{35, pdata.SeverityNumberINFO4},
		{entry.Notice, pdata.SeverityNumberINFO4},
		{45, pdata.SeverityNumberINFO4},
		{entry.Warning, pdata.SeverityNumberWARN},
		{51, pdata.SeverityNumberWARN},
		{entry.Warning2, pdata.SeverityNumberWARN2},
		{entry.Warning3, pdata.SeverityNumberWARN3},
		{entry.Warning4, pdata.SeverityNumberWARN4},
		{55, pdata.SeverityNumberWARN4},
		{entry.Error, pdata.SeverityNumberERROR},
		{61, pdata.SeverityNumberERROR},
		{entry.Error2, pdata.SeverityNumberERROR2},
		{entry.Error3, pdata.SeverityNumberERROR3},
		{entry.Error4, pdata.SeverityNumberERROR4},
		{65, pdata.SeverityNumberERROR4},
		{entry.Critical, pdata.SeverityNumberERROR4},
		{75, pdata.SeverityNumberERROR4},
		{entry.Alert, pdata.SeverityNumberERROR4},
		{85, pdata.SeverityNumberERROR4},
		{entry.Emergency, pdata.SeverityNumberFATAL},
		{91, pdata.SeverityNumberFATAL},
		{entry.Emergency2, pdata.SeverityNumberFATAL2},
		{entry.Emergency3, pdata.SeverityNumberFATAL3},
		{entry.Emergency4, pdata.SeverityNumberFATAL4},
		{95, pdata.SeverityNumberFATAL4},
		{entry.Catastrophe, pdata.SeverityNumberFATAL4},
	}

	for _, tc := range cases {
		t.Run(fmt.Sprintf("%v", tc.severity), func(t *testing.T) {
			entry := entry.New()
			entry.Severity = tc.severity
			log := convertAndDrill(entry)
			require.Equal(t, tc.expectedNumber, log.SeverityNumber())
		})
	}
}
