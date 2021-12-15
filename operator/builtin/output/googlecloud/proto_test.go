package googlecloud

import (
	"testing"

	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/structpb"
)

func TestToProto(t *testing.T) {
	testCases := []struct {
		name          string
		value         interface{}
		expectedValue *structpb.Value
		expectedErr   string
	}{
		{
			name:          "nil",
			value:         nil,
			expectedValue: structpb.NewNullValue(),
		},
		{
			name: "numbers",
			value: map[string]interface{}{
				"int":     int(1),
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
			},
			expectedValue: structpb.NewStructValue(&structpb.Struct{
				Fields: map[string]*structpb.Value{
					"int":     structpb.NewNumberValue(float64(1)),
					"int8":    structpb.NewNumberValue(float64(1)),
					"int16":   structpb.NewNumberValue(float64(1)),
					"int32":   structpb.NewNumberValue(float64(1)),
					"int64":   structpb.NewNumberValue(float64(1)),
					"uint":    structpb.NewNumberValue(float64(1)),
					"uint8":   structpb.NewNumberValue(float64(1)),
					"uint16":  structpb.NewNumberValue(float64(1)),
					"uint32":  structpb.NewNumberValue(float64(1)),
					"uint64":  structpb.NewNumberValue(float64(1)),
					"float32": structpb.NewNumberValue(float64(1)),
					"float64": structpb.NewNumberValue(float64(1)),
				},
			}),
		},
		{
			name: "map[string]string",
			value: map[string]string{
				"test": "value",
			},
			expectedValue: structpb.NewStructValue(&structpb.Struct{
				Fields: map[string]*structpb.Value{
					"test": structpb.NewStringValue("value"),
				},
			}),
		},
		{
			name:  "interface list",
			value: []interface{}{"value", 1},
			expectedValue: structpb.NewListValue(&structpb.ListValue{Values: []*structpb.Value{
				structpb.NewStringValue("value"),
				structpb.NewNumberValue(float64(1)),
			}}),
		},
		{
			name:  "string list",
			value: []string{"value", "test"},
			expectedValue: structpb.NewListValue(&structpb.ListValue{Values: []*structpb.Value{
				structpb.NewStringValue("value"),
				structpb.NewStringValue("test"),
			}}),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			proto, err := toProto(tc.value)
			if tc.expectedErr != "" {
				require.Error(t, err)
				require.Contains(t, err, tc.expectedErr)
				return
			}

			require.NoError(t, err)
			require.Equal(t, tc.expectedValue, proto)
		})
	}
}
