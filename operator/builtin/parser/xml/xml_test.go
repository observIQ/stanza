package xml

import (
	"context"
	"errors"
	"testing"

	"github.com/observiq/stanza/v2/operator"
	"github.com/observiq/stanza/v2/testutil"
	"github.com/open-telemetry/opentelemetry-log-collection/entry"
	"github.com/stretchr/testify/require"
)

func TestParse(t *testing.T) {
	var strictDisabled bool = false

	testCases := []struct {
		name           string
		value          interface{}
		strict         *bool
		expectedResult interface{}
		expectedErr    error
	}{
		{
			name:        "Empty string",
			value:       "",
			expectedErr: errors.New("failed to decode as xml"),
		},
		{
			name:        "No elements",
			value:       "regular string",
			expectedErr: errors.New("no xml elements found"),
		},
		{
			name:        "Non string value",
			value:       5,
			expectedErr: errors.New("value passed to parser is not a string"),
		},
		{
			name:        "Incomplete element",
			value:       "<person age='30'>",
			expectedErr: errors.New("failed to get next xml token"),
		},
		{
			name:        "Invalid start character",
			value:       "person age='30'></person>",
			expectedErr: errors.New("failed to get next xml token"),
		},
		{
			name:        "Invalid end character",
			value:       "<person age='30'></person",
			expectedErr: errors.New("failed to get next xml token"),
		},
		{
			name:        "Invalid attribute",
			value:       "<person age=30></person>",
			expectedErr: errors.New("unquoted or missing attribute value in element"),
		},
		{
			name:        "Not matching element",
			value:       "<person age='30'></dog>",
			expectedErr: errors.New("element <person> closed by </dog>"),
		},
		{
			name:  "Single element",
			value: "<person age='30'>Jon Smith</person>",
			expectedResult: map[string]interface{}{
				"tag": "person",
				"attributes": map[string]interface{}{
					"age": "30",
				},
				"content": "Jon Smith",
			},
		},
		{
			name:   "Special character without strict",
			value:  "<person company='at&t'>Jon Smith</person>",
			strict: &strictDisabled,
			expectedResult: map[string]interface{}{
				"tag": "person",
				"attributes": map[string]interface{}{
					"company": "at&t",
				},
				"content": "Jon Smith",
			},
		},
		{
			name:  "Multiple elements",
			value: "<person age='30'>Jon Smith</person><person age='28'>Sally Smith</person>",
			expectedResult: []map[string]interface{}{
				{
					"tag": "person",
					"attributes": map[string]interface{}{
						"age": "30",
					},
					"content": "Jon Smith",
				},
				{
					"tag": "person",
					"attributes": map[string]interface{}{
						"age": "28",
					},
					"content": "Sally Smith",
				},
			},
		},
		{
			name:  "Child elements",
			value: "<office><worker><person age='30'>Jon Smith</person></worker></office>",
			expectedResult: map[string]interface{}{
				"tag": "office",
				"children": []map[string]interface{}{
					{
						"tag": "worker",
						"children": []map[string]interface{}{
							{
								"tag": "person",
								"attributes": map[string]interface{}{
									"age": "30",
								},
								"content": "Jon Smith",
							},
						},
					},
				},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			config := NewXMLParserConfig("test")
			config.Strict = tc.strict

			ops, err := config.Build(testutil.NewBuildContext(t))
			op := ops[0]
			parser, ok := op.(*XMLParser)
			require.True(t, ok)

			result, err := parser.parse(tc.value)
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
		Body: "<test>test value</test>",
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
