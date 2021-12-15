package googlecloud

import (
	"encoding/base64"
	"fmt"
	"unicode/utf8"

	"google.golang.org/protobuf/types/known/structpb"
)

// toProto converts a value to a protobuf equivalent
func toProto(v interface{}) (*structpb.Value, error) {
	switch v := v.(type) {
	case nil:
		return structpb.NewNullValue(), nil
	case bool:
		return structpb.NewBoolValue(v), nil
	case int:
		return structpb.NewNumberValue(float64(v)), nil
	case int8:
		return structpb.NewNumberValue(float64(v)), nil
	case int16:
		return structpb.NewNumberValue(float64(v)), nil
	case int32:
		return structpb.NewNumberValue(float64(v)), nil
	case int64:
		return structpb.NewNumberValue(float64(v)), nil
	case uint:
		return structpb.NewNumberValue(float64(v)), nil
	case uint8:
		return structpb.NewNumberValue(float64(v)), nil
	case uint16:
		return structpb.NewNumberValue(float64(v)), nil
	case uint32:
		return structpb.NewNumberValue(float64(v)), nil
	case uint64:
		return structpb.NewNumberValue(float64(v)), nil
	case float32:
		return structpb.NewNumberValue(float64(v)), nil
	case float64:
		return structpb.NewNumberValue(v), nil
	case string:
		return toProtoString(v)
	case []byte:
		s := base64.StdEncoding.EncodeToString(v)
		return structpb.NewStringValue(s), nil
	case map[string]interface{}:
		v2, err := toProtoStruct(v)
		if err != nil {
			return nil, err
		}
		return structpb.NewStructValue(v2), nil
	case map[string]string:
		fields := map[string]*structpb.Value{}
		for key, value := range v {
			fields[key] = structpb.NewStringValue(value)
		}
		return structpb.NewStructValue(&structpb.Struct{
			Fields: fields,
		}), nil
	case []interface{}:
		v2, err := toProtoList(v)
		if err != nil {
			return nil, err
		}
		return structpb.NewListValue(v2), nil
	case []string:
		values := []*structpb.Value{}
		for _, str := range v {
			values = append(values, structpb.NewStringValue(str))
		}

		return structpb.NewListValue(&structpb.ListValue{
			Values: values,
		}), nil
	default:
		return nil, fmt.Errorf("invalid type: %T", v)
	}
}

// toProtoStruct converts a map to a protobuf equivalent
func toProtoStruct(v map[string]interface{}) (*structpb.Struct, error) {
	x := &structpb.Struct{Fields: make(map[string]*structpb.Value, len(v))}
	for k, v := range v {
		if !utf8.ValidString(k) {
			return nil, fmt.Errorf("invalid UTF-8 in string: %q", k)
		}
		var err error
		x.Fields[k], err = toProto(v)
		if err != nil {
			return nil, err
		}
	}
	return x, nil
}

// toProtoList converts a slice of interface a protobuf equivalent
func toProtoList(v []interface{}) (*structpb.ListValue, error) {
	x := &structpb.ListValue{Values: make([]*structpb.Value, len(v))}
	for i, v := range v {
		var err error
		x.Values[i], err = toProto(v)
		if err != nil {
			return nil, err
		}
	}
	return x, nil
}

// toProtoString converts a string to a proto string
func toProtoString(v string) (*structpb.Value, error) {
	if !utf8.ValidString(v) {
		return nil, fmt.Errorf("invalid UTF-8 in string: %q", v)
	}
	return structpb.NewStringValue(v), nil
}
