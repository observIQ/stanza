package otlp

import (
	"encoding/json"
	"fmt"

	"github.com/observiq/stanza/entry"
	"github.com/observiq/stanza/version"
	"go.opentelemetry.io/collector/consumer/pdata"
)

func convert(entries []*entry.Entry) pdata.Logs {

	out := pdata.NewLogs()
	logs := out.ResourceLogs()

	entriesByResource := groupByResource(entries)

	logs.Resize(len(entriesByResource))

	for i, resourceEntries := range entriesByResource {
		rls := logs.At(i)
		resource := rls.Resource()
		resource.InitEmpty()

		resourceAtts := resource.Attributes()
		for k, v := range resourceEntries[0].Resource {
			resourceAtts.InsertString(k, v)
		}

		rls.InstrumentationLibraryLogs().Resize(1)
		ills := rls.InstrumentationLibraryLogs().At(0)
		ills.InitEmpty()

		il := ills.InstrumentationLibrary()
		il.InitEmpty()
		il.SetName("stanza")
		il.SetVersion(version.GetVersion())

		for _, entry := range resourceEntries {
			lr := pdata.NewLogRecord()
			lr.InitEmpty()
			lr.SetTimestamp(pdata.TimestampUnixNano(entry.Timestamp.UnixNano()))

			sevText, sevNum := convertSeverity(entry.Severity)
			lr.SetSeverityText(sevText)
			lr.SetSeverityNumber(sevNum)

			if len(entry.Labels) > 0 {
				attributes := lr.Attributes()
				for k, v := range entry.Labels {
					attributes.InsertString(k, v)
				}
			}

			lr.Body().InitEmpty()
			insertToAttributeVal(entry.Record, lr.Body())

			ills.Logs().Append(lr)
		}
	}

	return out
}

func groupByResource(entries []*entry.Entry) [][]*entry.Entry {
	resourceMap := make(map[string][]*entry.Entry)

	for _, ent := range entries {
		resourceBytes, err := json.Marshal(ent.Resource)
		if err != nil {
			continue // not expected to ever happen
		}
		resourceHash := string(resourceBytes)

		if resourceEntries, ok := resourceMap[resourceHash]; ok {
			resourceEntries = append(resourceEntries, ent)
		} else {
			resourceMap[resourceHash] = []*entry.Entry{ent}
		}
	}

	entriesByResource := make([][]*entry.Entry, 0, len(resourceMap))
	for _, v := range resourceMap {
		entriesByResource = append(entriesByResource, v)
	}
	return entriesByResource
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
	case map[string]interface{}:
		dest.SetMapVal(toAttributeMap(t))
	case []interface{}:
		dest.SetArrayVal(toAttributeArray(t))
	default:
		dest.SetStringVal(fmt.Sprintf("%v", t))
	}
}

func toAttributeMap(obsMap map[string]interface{}) pdata.AttributeMap {
	attMap := pdata.NewAttributeMap()
	attMap.InitEmptyWithCapacity(len(obsMap))
	for k, v := range obsMap {
		switch t := v.(type) {
		case bool:
			attMap.InsertBool(k, t)
		case string:
			attMap.InsertString(k, t)
		case []byte:
			attMap.InsertString(k, string(t))
		case int64:
			attMap.InsertInt(k, t)
		case int32:
			attMap.InsertInt(k, int64(t))
		case int16:
			attMap.InsertInt(k, int64(t))
		case int8:
			attMap.InsertInt(k, int64(t))
		case int:
			attMap.InsertInt(k, int64(t))
		case uint64:
			attMap.InsertInt(k, int64(t))
		case uint32:
			attMap.InsertInt(k, int64(t))
		case uint16:
			attMap.InsertInt(k, int64(t))
		case uint8:
			attMap.InsertInt(k, int64(t))
		case uint:
			attMap.InsertInt(k, int64(t))
		case float64:
			attMap.InsertDouble(k, t)
		case float32:
			attMap.InsertDouble(k, float64(t))
		case map[string]interface{}:
			subMap := toAttributeMap(t)
			subMapVal := pdata.NewAttributeValueMap()
			subMapVal.SetMapVal(subMap)
			attMap.Insert(k, subMapVal)
		case []interface{}:
			arr := toAttributeArray(t)
			arrVal := pdata.NewAttributeValueArray()
			arrVal.SetArrayVal(arr)
			attMap.Insert(k, arrVal)
		default:
			attMap.InsertString(k, fmt.Sprintf("%v", t))
		}
	}
	return attMap
}

func toAttributeArray(obsArr []interface{}) pdata.AnyValueArray {
	arr := pdata.NewAnyValueArray()
	for _, v := range obsArr {
		attVal := pdata.NewAttributeValue()
		insertToAttributeVal(v, attVal)
		arr.Append(attVal)
	}
	return arr
}

func convertSeverity(s entry.Severity) (string, pdata.SeverityNumber) {
	switch {

	// Handle standard severity levels
	case s == entry.Catastrophe:
		return "Fatal", pdata.SeverityNumberFATAL4
	case s == entry.Emergency:
		return "Error", pdata.SeverityNumberFATAL
	case s == entry.Alert:
		return "Error", pdata.SeverityNumberERROR3
	case s == entry.Critical:
		return "Error", pdata.SeverityNumberERROR2
	case s == entry.Error:
		return "Error", pdata.SeverityNumberERROR
	case s == entry.Warning:
		return "Info", pdata.SeverityNumberINFO4
	case s == entry.Notice:
		return "Info", pdata.SeverityNumberINFO3
	case s == entry.Info:
		return "Info", pdata.SeverityNumberINFO
	case s == entry.Debug:
		return "Debug", pdata.SeverityNumberDEBUG
	case s == entry.Trace:
		return "Trace", pdata.SeverityNumberTRACE2

	// Handle custom severity levels
	case s > entry.Emergency:
		return "Fatal", pdata.SeverityNumberFATAL2
	case s > entry.Alert:
		return "Error", pdata.SeverityNumberERROR4
	case s > entry.Critical:
		return "Error", pdata.SeverityNumberERROR3
	case s > entry.Error:
		return "Error", pdata.SeverityNumberERROR2
	case s > entry.Warning:
		return "Info", pdata.SeverityNumberINFO4
	case s > entry.Notice:
		return "Info", pdata.SeverityNumberINFO3
	case s > entry.Info:
		return "Info", pdata.SeverityNumberINFO2
	case s > entry.Debug:
		return "Debug", pdata.SeverityNumberDEBUG2
	case s > entry.Trace:
		return "Trace", pdata.SeverityNumberTRACE3
	case s > entry.Default:
		return "Trace", pdata.SeverityNumberTRACE

	default:
		return "Undefined", pdata.SeverityNumberUNDEFINED
	}
}
