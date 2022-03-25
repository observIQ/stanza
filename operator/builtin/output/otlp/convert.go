package otlp

import (
	"fmt"

	"github.com/observiq/stanza/entry"
	"go.opentelemetry.io/collector/model/pdata"
)

// Convert converts entry.Entry into provided pdata.LogRecord.
func convert(ent *entry.Entry) pdata.Logs {
	pLogs := pdata.NewLogs()
	logs := pLogs.ResourceLogs()

	rls := logs.AppendEmpty()

	resource := rls.Resource()
	insertToAttributeMap(ent.Resource, resource.Attributes())

	ills := rls.InstrumentationLibraryLogs().AppendEmpty()
	lr := ills.LogRecords().AppendEmpty()
	convertInto(ent, lr)
	return pLogs
}

func convertInto(ent *entry.Entry, dest pdata.LogRecord) {
	dest.SetTimestamp(pdata.NewTimestampFromTime(ent.Timestamp))
	dest.SetSeverityNumber(sevMap[ent.Severity])
	dest.SetSeverityText(sevTextMap[ent.Severity])
	insertToAttributeVal(ent.Record, dest.Body())
}

func insertToAttributeVal(value interface{}, dest pdata.AttributeValue) {
	switch t := value.(type) {
	case bool:
		dest.SetBoolVal(t)
	case string:
		dest.SetStringVal(t)
	case []byte:
		dest.SetStringVal(string(t))
	case int64:
		dest.SetIntVal(t)
	case int32:
		dest.SetIntVal(int64(t))
	case int16:
		dest.SetIntVal(int64(t))
	case int8:
		dest.SetIntVal(int64(t))
	case int:
		dest.SetIntVal(int64(t))
	case uint64:
		dest.SetIntVal(int64(t))
	case uint32:
		dest.SetIntVal(int64(t))
	case uint16:
		dest.SetIntVal(int64(t))
	case uint8:
		dest.SetIntVal(int64(t))
	case uint:
		dest.SetIntVal(int64(t))
	case float64:
		dest.SetDoubleVal(t)
	case float32:
		dest.SetDoubleVal(float64(t))
	case []interface{}:
		toAttributeArray(t).CopyTo(dest)
	default:
		dest.SetStringVal(fmt.Sprintf("%v", t))
	}
}

func insertToAttributeMap(obsMap map[string]string, dest pdata.AttributeMap) {
	dest.EnsureCapacity(len(obsMap))
	for k, v := range obsMap {
		dest.InsertString(k, v)
	}
}

func toAttributeArray(obsArr []interface{}) pdata.AttributeValue {
	arrVal := pdata.NewAttributeValueArray()
	arr := arrVal.SliceVal()
	arr.EnsureCapacity(len(obsArr))
	for _, v := range obsArr {
		insertToAttributeVal(v, arr.AppendEmpty())
	}
	return arrVal
}

var sevMap = map[entry.Severity]pdata.SeverityNumber{
	entry.Default: pdata.SeverityNumberUNDEFINED,
	entry.Trace:   pdata.SeverityNumberTRACE,
	entry.Trace2:  pdata.SeverityNumberTRACE2,
	entry.Trace3:  pdata.SeverityNumberTRACE3,
	entry.Trace4:  pdata.SeverityNumberTRACE4,
	entry.Debug:   pdata.SeverityNumberDEBUG,
	entry.Debug2:  pdata.SeverityNumberDEBUG2,
	entry.Debug3:  pdata.SeverityNumberDEBUG3,
	entry.Debug4:  pdata.SeverityNumberDEBUG4,
	entry.Info:    pdata.SeverityNumberINFO,
	entry.Info2:   pdata.SeverityNumberINFO2,
	entry.Info3:   pdata.SeverityNumberINFO3,
	entry.Info4:   pdata.SeverityNumberINFO4,
	entry.Error2:  pdata.SeverityNumberERROR2,
	entry.Error3:  pdata.SeverityNumberERROR3,
	entry.Error4:  pdata.SeverityNumberERROR4,
}

var sevTextMap = map[entry.Severity]string{
	entry.Default: "",
	entry.Trace:   "Trace",
	entry.Trace2:  "Trace2",
	entry.Trace3:  "Trace3",
	entry.Trace4:  "Trace4",
	entry.Debug:   "Debug",
	entry.Debug2:  "Debug2",
	entry.Debug3:  "Debug3",
	entry.Debug4:  "Debug4",
	entry.Info:    "Info",
	entry.Info2:   "Info2",
	entry.Info3:   "Info3",
	entry.Info4:   "Info4",
	entry.Error:   "Error",
	entry.Error2:  "Error2",
	entry.Error3:  "Error3",
	entry.Error4:  "Error4",
}
