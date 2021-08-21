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

func newTestParser(t *testing.T) *CSVParser {
	cfg := NewCSVParserConfig("test")
	cfg.Header = testHeader
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

func TestCSVParserBuildFailureInvalidDelimiter(t *testing.T) {
	cfg := NewCSVParserConfig("test")
	cfg.Header = testHeader
	cfg.FieldDelimiter = ";;"
	_, err := cfg.Build(testutil.NewBuildContext(t))
	require.Error(t, err)
	require.Contains(t, err.Error(), "Invalid 'delimiter': ';;'")
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
		inputEntry   entry.Entry
		outputRecord interface{}
	}{
		{
			"basic",
			func(p *CSVParserConfig) {
				p.Header = testHeader
			},
			entry.Entry{
				Record: "stanza,INFO,started agent",
			},
			map[string]interface{}{
				"name": "stanza",
				"sev":  "INFO",
				"msg":  "started agent",
			},
		},
		{
			"advanced",
			func(p *CSVParserConfig) {
				p.Header = "name;address;age;phone;position"
				p.FieldDelimiter = ";"
			},
			entry.Entry{
				Record: "stanza;Evergreen;1;555-5555;agent",
			},
			map[string]interface{}{
				"name":     "stanza",
				"address":  "Evergreen",
				"age":      "1",
				"phone":    "555-5555",
				"position": "agent",
			},
		},
		{
			"mariadb-audit-log",
			func(p *CSVParserConfig) {
				p.Header = "timestamp,serverhost,username,host,connectionid,queryid,operation,database,object,retcode"
			},
			entry.Entry{
				Record: "20210316 17:08:01,oiq-int-mysql,load,oiq-int-mysql.bluemedora.localnet,5,0,DISCONNECT,,,0",
			},
			map[string]interface{}{
				"timestamp":    "20210316 17:08:01",
				"serverhost":   "oiq-int-mysql",
				"username":     "load",
				"host":         "oiq-int-mysql.bluemedora.localnet",
				"connectionid": "5",
				"queryid":      "0",
				"operation":    "DISCONNECT",
				"database":     "",
				"object":       "",
				"retcode":      "0",
			},
		},
		{
			"empty field",
			func(p *CSVParserConfig) {
				p.Header = "name,address,age,phone,position"
			},
			entry.Entry{
				Record: "stanza,Evergreen,,555-5555,agent",
			},
			map[string]interface{}{
				"name":     "stanza",
				"address":  "Evergreen",
				"age":      "",
				"phone":    "555-5555",
				"position": "agent",
			},
		},
		{
			"tab delimiter",
			func(p *CSVParserConfig) {
				p.Header = "name	address	age	phone	position"
				p.FieldDelimiter = "\t"
			},
			entry.Entry{
				Record: "stanza	Evergreen	1	555-5555	agent",
			},
			map[string]interface{}{
				"name":     "stanza",
				"address":  "Evergreen",
				"age":      "1",
				"phone":    "555-5555",
				"position": "agent",
			},
		},
		{
			"comma in quotes",
			func(p *CSVParserConfig) {
				p.Header = "name,address,age,phone,position"
			},
			entry.Entry{
				Record: "stanza,\"Evergreen,49508\",1,555-5555,agent",
			},
			map[string]interface{}{
				"name":     "stanza",
				"address":  "Evergreen,49508",
				"age":      "1",
				"phone":    "555-5555",
				"position": "agent",
			},
		},
		{
			"quotes in quotes",
			func(p *CSVParserConfig) {
				p.Header = "name,address,age,phone,position"
			},
			entry.Entry{
				Record: "\"bob \"\"the man\"\"\",Evergreen,1,555-5555,agent",
			},
			map[string]interface{}{
				"name":     "bob \"the man\"",
				"address":  "Evergreen",
				"age":      "1",
				"phone":    "555-5555",
				"position": "agent",
			},
		},
		{
			"header-delimiter",
			func(p *CSVParserConfig) {
				p.Header = "name+sev+msg"
				p.HeaderDelimiter = "+"
			},
			entry.Entry{
				Record: "stanza,INFO,started agent",
			},
			map[string]interface{}{
				"name": "stanza",
				"sev":  "INFO",
				"msg":  "started agent",
			},
		},
		{
			"tab-delimiter",
			func(p *CSVParserConfig) {
				p.Header = testHeader
				p.HeaderDelimiter = ","
				p.FieldDelimiter = "\t"
			},
			entry.Entry{
				Record: "stanza\tINFO\tstarted agent",
			},
			map[string]interface{}{
				"name": "stanza",
				"sev":  "INFO",
				"msg":  "started agent",
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

			err = op.Process(context.Background(), &tc.inputEntry)
			require.NoError(t, err)

			fake.ExpectRecord(t, tc.outputRecord)
		})
	}
}

func TestParserCSVMultipleRecords(t *testing.T) {
	t.Run("basic", func(t *testing.T) {
		cfg := NewCSVParserConfig("test")
		cfg.OutputIDs = []string{"fake"}
		cfg.Header = testHeader

		ops, err := cfg.Build(testutil.NewBuildContext(t))
		require.NoError(t, err)
		op := ops[0]

		fake := testutil.NewFakeOutput(t)
		op.SetOutputs([]operator.Operator{fake})

		entry := entry.New()
		entry.Record = "stanza,INFO,started agent\nstanza,DEBUG,started agent"
		err = op.Process(context.Background(), entry)
		// require.Nil(t, err, "Expected to parse a single csv record, got '2'")
		// require.Contains(t, err.Error(), "Expected to parse a single csv record, got '2'")
		require.NoError(t, err)

		fake.ExpectRecord(t, map[string]interface{}{
			"name": "stanza",
			"sev":  "DEBUG",
			"msg":  "started agent",
		})
	})
}

func TestParserCSVInvalidJSONInput(t *testing.T) {
	t.Run("basic", func(t *testing.T) {
		cfg := NewCSVParserConfig("test")
		cfg.OutputIDs = []string{"fake"}
		cfg.Header = testHeader

		ops, err := cfg.Build(testutil.NewBuildContext(t))
		require.NoError(t, err)
		op := ops[0]

		fake := testutil.NewFakeOutput(t)
		op.SetOutputs([]operator.Operator{fake})

		entry := entry.New()
		entry.Record = "{\"name\": \"stanza\"}"
		err = op.Process(context.Background(), entry)
		require.Error(t, err, "parse error on line 1, column 1: bare \" in non-quoted-field")
		fake.ExpectRecord(t, "{\"name\": \"stanza\"}")
	})
}

func TestBuildParserCSV(t *testing.T) {
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
		require.Contains(t, err.Error(), "missing header delimiter in header")
	})

	t.Run("InvalidHeaderFieldWrongDelimiter", func(t *testing.T) {
		c := newBasicCSVParser()
		c.Header = "name;position;number"
		_, err := c.Build(testutil.NewBuildContext(t))
		require.Error(t, err)
	})

	t.Run("InvalidDelimiter", func(t *testing.T) {
		c := newBasicCSVParser()
		c.Header = "name,position,number"
		c.FieldDelimiter = ":"
		_, err := c.Build(testutil.NewBuildContext(t))
		require.Error(t, err)
		require.Contains(t, err.Error(), "missing header delimiter in header")
	})

	t.Run("HeaderDelimiter", func(t *testing.T) {
		c := newBasicCSVParser()
		c.Header = "name+position+number"
		c.HeaderDelimiter = "+"
		c.FieldDelimiter = ":"
		_, err := c.Build(testutil.NewBuildContext(t))
		require.NoError(t, err)
	})

	t.Run("InvalidHeaderDelimiter", func(t *testing.T) {
		c := newBasicCSVParser()
		c.Header = "name,position,number"
		c.HeaderDelimiter = "+"
		c.FieldDelimiter = ":"
		_, err := c.Build(testutil.NewBuildContext(t))
		require.Error(t, err)
		require.Contains(t, err.Error(), "missing header delimiter in header")
	})
}
