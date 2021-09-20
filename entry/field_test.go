package entry

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
	yaml "gopkg.in/yaml.v2"
)

func TestFieldUnmarshalJSON(t *testing.T) {
	cases := []struct {
		name     string
		input    []byte
		expected Field
	}{
		{
			"SimpleField",
			[]byte(`"test1"`),
			NewBodyField("test1"),
		},
		{
			"ComplexField",
			[]byte(`"test1.test2"`),
			NewBodyField("test1", "test2"),
		},
		{
			"RootField",
			[]byte(`"$"`),
			NewBodyField([]string{}...),
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var f Field
			err := json.Unmarshal(tc.input, &f)
			require.NoError(t, err)

			require.Equal(t, tc.expected, f)
		})
	}
}

func TestFieldUnmarshalJSONFailure(t *testing.T) {
	invalidField := []byte(`{"key":"value"}`)
	var f Field
	err := json.Unmarshal(invalidField, &f)
	require.Error(t, err)
	require.Contains(t, err.Error(), "cannot unmarshal object into Go value of type string")
}

func TestFieldMarshalJSON(t *testing.T) {
	cases := []struct {
		name     string
		input    Field
		expected []byte
	}{
		{
			"SimpleField",
			NewBodyField("test1"),
			[]byte(`"test1"`),
		},
		{
			"ComplexField",
			NewBodyField("test1", "test2"),
			[]byte(`"test1.test2"`),
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			res, err := json.Marshal(tc.input)
			require.NoError(t, err)

			require.Equal(t, tc.expected, res)
		})
	}
}

func TestFieldUnmarshalYAML(t *testing.T) {
	cases := []struct {
		name     string
		input    []byte
		expected Field
	}{
		{
			"SimpleField",
			[]byte(`"test1"`),
			NewBodyField("test1"),
		},
		{
			"UnquotedField",
			[]byte(`test1`),
			NewBodyField("test1"),
		},
		{
			"RootField",
			[]byte(`"$"`),
			NewBodyField([]string{}...),
		},
		{
			"ComplexField",
			[]byte(`"test1.test2"`),
			NewBodyField("test1", "test2"),
		},
		{
			"ComplexFieldWithRoot",
			[]byte(`"$.test1.test2"`),
			NewBodyField("test1", "test2"),
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var f Field
			err := yaml.UnmarshalStrict(tc.input, &f)
			require.NoError(t, err)

			require.Equal(t, tc.expected, f)
		})
	}
}

func TestFieldUnmarshalYAMLFailure(t *testing.T) {
	invalidField := []byte(`invalid: field`)
	var f Field
	err := yaml.UnmarshalStrict(invalidField, &f)
	require.Error(t, err)
	require.Contains(t, err.Error(), "cannot unmarshal !!map into string")
}

func TestFieldMarshalYAML(t *testing.T) {
	cases := []struct {
		name     string
		input    interface{}
		expected string
	}{
		{
			"SimpleField",
			NewBodyField("test1"),
			"test1\n",
		},
		{
			"ComplexField",
			NewBodyField("test1", "test2"),
			"test1.test2\n",
		},
		{
			"EmptyField",
			NewBodyField(),
			"$body\n",
		},
		{
			"FieldWithDots",
			NewBodyField("test.1"),
			"$body['test.1']\n",
		},
		{
			"FieldWithDotsThenNone",
			NewBodyField("test.1", "test2"),
			"$body['test.1']['test2']\n",
		},
		{
			"FieldWithNoDotsThenDots",
			NewBodyField("test1", "test.2"),
			"$body['test1']['test.2']\n",
		},
		{
			"AttributeField",
			NewAttributeField("test1"),
			"$attributes.test1\n",
		},
		{
			"AttributeFieldWithDots",
			NewAttributeField("test.1"),
			"$attributes['test.1']\n",
		},
		{
			"ResourceField",
			NewResourceField("test1"),
			"$resource.test1\n",
		},
		{
			"ResourceFieldWithDots",
			NewResourceField("test.1"),
			"$resource['test.1']\n",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			res, err := yaml.Marshal(tc.input)
			require.NoError(t, err)

			require.Equal(t, tc.expected, string(res))
		})
	}
}

func TestSplitField(t *testing.T) {
	cases := []struct {
		name      string
		input     string
		output    []string
		expectErr bool
	}{
		{"Simple", "test", []string{"test"}, false},
		{"Sub", "test.case", []string{"test", "case"}, false},
		{"Root", "$", []string{"$"}, false},
		{"RootWithSub", "$body.field", []string{"$body", "field"}, false},
		{"RootWithTwoSub", "$body.field1.field2", []string{"$body", "field1", "field2"}, false},
		{"BracketSyntaxSingleQuote", "['test']", []string{"test"}, false},
		{"BracketSyntaxDoubleQuote", `["test"]`, []string{"test"}, false},
		{"RootSubBracketSyntax", `$body["test"]`, []string{"$body", "test"}, false},
		{"BracketThenDot", `$body["test1"].test2`, []string{"$body", "test1", "test2"}, false},
		{"BracketThenBracket", `$body["test1"]["test2"]`, []string{"$body", "test1", "test2"}, false},
		{"DotThenBracket", `$body.test1["test2"]`, []string{"$body", "test1", "test2"}, false},
		{"DotsInBrackets", `$body["test1.test2"]`, []string{"$body", "test1.test2"}, false},
		{"UnclosedBrackets", `$body["test1.test2"`, nil, true},
		{"UnclosedQuotes", `$body["test1.test2]`, nil, true},
		{"UnmatchedQuotes", `$body["test1.test2']`, nil, true},
		{"BracketAtEnd", `$body[`, nil, true},
		{"SingleQuoteAtEnd", `$body['`, nil, true},
		{"DoubleQuoteAtEnd", `$body["`, nil, true},
		{"BracketMissingQuotes", `$body[test]`, nil, true},
		{"CharacterBetweenBracketAndQuote", `$body["test"a]`, nil, true},
		{"CharacterOutsideBracket", `$body["test"]a`, nil, true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			s, err := splitField(tc.input)
			if tc.expectErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)

			require.Equal(t, tc.output, s)
		})
	}
}

func TestFieldFromStringInvalidSplit(t *testing.T) {
	_, err := fieldFromString("$resource[test]")
	require.Error(t, err)
	require.Contains(t, err.Error(), "splitting field")
}

func TestFieldFromStringWithResource(t *testing.T) {
	field, err := fieldFromString(`$resource["test"]`)
	require.NoError(t, err)
	require.Equal(t, "$resource.test", field.String())
}

func TestFieldFromStringWithInvalidResource(t *testing.T) {
	_, err := fieldFromString(`$resource["test"]["key"]`)
	require.Error(t, err)
	require.Contains(t, err.Error(), "resource fields cannot be nested")
}
