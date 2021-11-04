package xml

import (
	"context"
	"errors"
	"testing"

	"github.com/observiq/stanza/entry"
	"github.com/observiq/stanza/operator"
	"github.com/observiq/stanza/testutil"
	"github.com/stretchr/testify/require"
)

func TestParse(t *testing.T) {
	testCases := []struct {
		name           string
		value          interface{}
		expectedResult interface{}
		expectedErr    error
	}{
		{
			name:        "Empty string",
			value:       "",
			expectedErr: errors.New("failed to decode as xml"),
		},
		{
			name:        "No nodes",
			value:       "regular string",
			expectedErr: errors.New("no xml nodes found"),
		},
		{
			name:        "Non string value",
			value:       5,
			expectedErr: errors.New("value is not a string"),
		},
		{
			name:        "Incomplete node",
			value:       "<person age='30'>",
			expectedErr: errors.New("failed to get next xml token"),
		},
		{
			name:  "Single node",
			value: "<person age='30'>Jon Smith</person>",
			expectedResult: map[string]interface{}{
				"type": "person",
				"attributes": map[string]string{
					"age": "30",
				},
				"value": "Jon Smith",
			},
		},
		{
			name:  "Multiple nodes",
			value: "<person age='30'>Jon Smith</person><person age='28'>Sally Smith</person>",
			expectedResult: []map[string]interface{}{
				{
					"type": "person",
					"attributes": map[string]string{
						"age": "30",
					},
					"value": "Jon Smith",
				},
				{
					"type": "person",
					"attributes": map[string]string{
						"age": "28",
					},
					"value": "Sally Smith",
				},
			},
		},
		{
			name:  "Children nodes",
			value: "<office><worker><person age='30'>Jon Smith</person></worker></office>",
			expectedResult: map[string]interface{}{
				"type": "office",
				"children": []map[string]interface{}{
					{
						"type": "worker",
						"children": []map[string]interface{}{
							{
								"type": "person",
								"attributes": map[string]string{
									"age": "30",
								},
								"value": "Jon Smith",
							},
						},
					},
				},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result, err := parse(tc.value)
			if tc.expectedErr != nil {
				require.Error(t, err)
				require.Contains(t, err.Error(), tc.expectedErr.Error())
			} else {
				require.NoError(t, err)
				require.Equal(t, tc.expectedResult, result)
			}
		})
	}
}

func TestXMLParserConfigBuild(t *testing.T) {
	config := NewXMLParserConfig("test")
	ops, err := config.Build(testutil.NewBuildContext(t))
	op := ops[0]
	require.NoError(t, err)
	require.IsType(t, &XMLParser{}, op)
}

func TestXMLParserConfigBuildFailure(t *testing.T) {
	config := NewXMLParserConfig("test")
	config.OnError = "invalid_on_error"
	_, err := config.Build(testutil.NewBuildContext(t))
	require.Error(t, err)
	require.Contains(t, err.Error(), "invalid `on_error` field")
}

func TestXMLPaserProcess(t *testing.T) {
	config := NewXMLParserConfig("test")
	ops, err := config.Build(testutil.NewBuildContext(t))
	require.NoError(t, err)

	op := ops[0]
	entry := &entry.Entry{
		Record: "<test>test value</test>",
	}

	err = op.Process(context.Background(), entry)
	require.NoError(t, err)
}

func TestXMLParserInitHook(t *testing.T) {
	builderFunc, ok := operator.DefaultRegistry.Lookup("xml_parser")
	require.True(t, ok)

	config := builderFunc()
	_, ok = config.(*XMLParserConfig)
	require.True(t, ok)
}
