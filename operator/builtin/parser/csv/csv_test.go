package csv

import (
	"context"
	"testing"

	"github.com/observiq/stanza/entry"
	"github.com/observiq/stanza/operator"
	"github.com/observiq/stanza/testutil"
	"github.com/stretchr/testify/require"
)

var testHeader = "name,sev,msg"
var testDelimiter = ","

func newTestParser(t *testing.T) *CSVParser {
	cfg := NewCSVParserConfig("test")
	cfg.Header = testHeader
  cfg.FieldDelimiter = testDelimiter
	ops, err := cfg.Build(testutil.NewBuildContext(t))
	require.NoError(t, err)
	op := ops[0]
	return op.(*CSVParser)
}

func TestCSVParserBuildFailure(t *testing.T) {
	cfg := NewCSVParserConfig("test")
	cfg.OnError = "invalid_on_error"
	_, err := cfg.Build(testutil.NewBuildContext(t))
	require.Error(t, err)
	require.Contains(t, err.Error(), "invalid `on_error` field")
}

func TestCSVParserStringFailure(t *testing.T) {
	parser := newTestParser(t)
	_, err := parser.parse("invalid")
	require.Error(t, err)
	require.Contains(t, err.Error(), "record on line 1: wrong number of fields")
}

func TestCSVParserByteFailure(t *testing.T) {
  parser := newTestParser(t)
	_, err := parser.parse([]byte("invalid"))
	require.Error(t, err)
  require.Contains(t, err.Error(), "record on line 1: wrong number of fields")
}

func TestCSVParserInvalidType(t *testing.T) {
  parser := newTestParser(t)
	_, err := parser.parse([]int{})
	require.Error(t, err)
	require.Contains(t, err.Error(), "type '[]int' cannot be parsed as csv")
}

func TestParserCSV(t *testing.T) {
	cases := []struct {
		name         string
		configure    func(*CSVParserConfig)
		inputRecord  interface{}
		outputRecord interface{}
	}{
		{
			"basic",
			func(p *CSVParserConfig) {
				p.Header = testHeader
        p.FieldDelimiter = testDelimiter
			},
			"stanza,INFO,started agent",
			map[string]interface{}{
				"name": "stanza",
        "sev": "INFO",
        "msg": "started agent",
			},
		},
    {
      "advanced",
      func(p *CSVParserConfig) {
        p.Header = "name;address;age;phone;position"
        p.FieldDelimiter = ";"
      },
      "stanza;Evergreen;1;555-5555;agent",
      map[string]interface{}{
        "name": "stanza",
        "address": "Evergreen",
        "age": "1",
        "phone": "555-5555",
        "position": "agent",
      },
    },
		{
      "empty field",
      func(p *CSVParserConfig) {
        p.Header = "name,address,age,phone,position"
        p.FieldDelimiter = testDelimiter
      },
      "stanza,Evergreen,,555-5555,agent",
      map[string]interface{}{
        "name": "stanza",
        "address": "Evergreen",
        "age": "",
        "phone": "555-5555",
        "position": "agent",
      },
    },
		{
      "tab delimiter",
      func(p *CSVParserConfig) {
        p.Header = "name	address	age	phone	position"
        p.FieldDelimiter = "\t"
      },
      "stanza	Evergreen	1	555-5555	agent",
      map[string]interface{}{
        "name": "stanza",
        "address": "Evergreen",
        "age": "1",
        "phone": "555-5555",
        "position": "agent",
      },
    },
		{
      "comma in quotes",
      func(p *CSVParserConfig) {
        p.Header = "name,address,age,phone,position"
        p.FieldDelimiter = testDelimiter
      },
      "stanza,\"Evergreen,49508\",1,555-5555,agent",
      map[string]interface{}{
        "name": "stanza",
        "address": "Evergreen,49508",
        "age": "1",
        "phone": "555-5555",
        "position": "agent",
      },
    },
		{
      "quotes in quotes",
      func(p *CSVParserConfig) {
        p.Header = "name,address,age,phone,position"
        p.FieldDelimiter = testDelimiter
      },
      "\"bob \"\"the man\"\"\",Evergreen,1,555-5555,agent",
      map[string]interface{}{
        "name": "bob \"the man\"",
        "address": "Evergreen",
        "age": "1",
        "phone": "555-5555",
        "position": "agent",
      },
    },
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			cfg := NewCSVParserConfig("test")
			cfg.OutputIDs = []string{"fake"}
			tc.configure(cfg)

			ops, err := cfg.Build(testutil.NewBuildContext(t))
			require.NoError(t, err)
			op := ops[0]

			fake := testutil.NewFakeOutput(t)
			op.SetOutputs([]operator.Operator{fake})

			entry := entry.New()
			entry.Record = tc.inputRecord
			err = op.Process(context.Background(), entry)
			require.NoError(t, err)

			fake.ExpectRecord(t, tc.outputRecord)
		})
	}
}

func TestBuildParserRegex(t *testing.T) {
	newBasicCSVParser := func() *CSVParserConfig {
		cfg := NewCSVParserConfig("test")
		cfg.OutputIDs = []string{"test"}
		cfg.Header = "name,position,number"
    cfg.FieldDelimiter = ","
		return cfg
	}

	t.Run("BasicConfig", func(t *testing.T) {
		c := newBasicCSVParser()
		_, err := c.Build(testutil.NewBuildContext(t))
		require.NoError(t, err)
	})

	t.Run("MissingHeaderField", func(t *testing.T) {
		c := newBasicCSVParser()
		c.Header = ""
		_, err := c.Build(testutil.NewBuildContext(t))
		require.Error(t, err)
	})

	t.Run("InvalidHeaderFieldMissingDelimiter", func(t *testing.T) {
		c := newBasicCSVParser()
		c.Header = "name"
		_, err := c.Build(testutil.NewBuildContext(t))
		require.Error(t, err)
	})

  t.Run("InvalidHeaderFieldWrongDelimiter", func(t *testing.T) {
    c := newBasicCSVParser()
    c.Header = "name;position;number"
    _, err := c.Build(testutil.NewBuildContext(t))
    require.Error(t, err)
  })

	/*t.Run("NoNamedGroups", func(t *testing.T) {
		c := newBasicCSVParser()
		c.Regex = ".*"
		_, err := c.Build(testutil.NewBuildContext(t))
		require.Error(t, err)
		require.Contains(t, err.Error(), "no named capture groups")
	})

	t.Run("NoNamedGroups", func(t *testing.T) {
		c := newBasicCSVParser()
		c.Regex = "(.*)"
		_, err := c.Build(testutil.NewBuildContext(t))
		require.Error(t, err)
		require.Contains(t, err.Error(), "no named capture groups")
	})*/
}
