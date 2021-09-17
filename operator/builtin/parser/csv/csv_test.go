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
		name             string
		configure        func(*CSVParserConfig)
		inputEntry       []entry.Entry
		outputBody       []interface{}
		expectBuildErr   bool
		expectProcessErr bool
	}{
		{
			"basic",
			func(p *CSVParserConfig) {
				p.Header = testHeader
			},
			[]entry.Entry{
				{
					Body: "stanza,INFO,started agent",
				},
			},
			[]interface{}{
				map[string]interface{}{
					"name": "stanza",
					"sev":  "INFO",
					"msg":  "started agent",
				},
			},
			false,
			false,
		},
		{
			"basic-multiple-static-bodys",
			func(p *CSVParserConfig) {
				p.Header = testHeader
			},
			[]entry.Entry{
				{
					Body: "stanza,INFO,started agent",
				},
				{
					Body: "stanza,ERROR,agent killed",
				},
				{
					Body: "kernel,TRACE,oom",
				},
			},
			[]interface{}{
				map[string]interface{}{
					"name": "stanza",
					"sev":  "INFO",
					"msg":  "started agent",
				},
				map[string]interface{}{
					"name": "stanza",
					"sev":  "ERROR",
					"msg":  "agent killed",
				},
				map[string]interface{}{
					"name": "kernel",
					"sev":  "TRACE",
					"msg":  "oom",
				},
			},
			false,
			false,
		},
		{
			"advanced",
			func(p *CSVParserConfig) {
				p.Header = "name;address;age;phone;position"
				p.FieldDelimiter = ";"
			},
			[]entry.Entry{
				{
					Body: "stanza;Evergreen;1;555-5555;agent",
				},
			},
			[]interface{}{
				map[string]interface{}{
					"name":     "stanza",
					"address":  "Evergreen",
					"age":      "1",
					"phone":    "555-5555",
					"position": "agent",
				},
			},
			false,
			false,
		},
		{
			"dynamic-fields",
			func(p *CSVParserConfig) {
				p.HeaderLabel = "Fields"
				p.FieldDelimiter = ","
			},
			[]entry.Entry{
				{
					Labels: map[string]string{
						"Fields": "name,age,height,number",
					},
					Body: "stanza dev,1,400,555-555-5555",
				},
			},
			[]interface{}{
				map[string]interface{}{
					"name":   "stanza dev",
					"age":    "1",
					"height": "400",
					"number": "555-555-5555",
				},
			},
			false,
			false,
		},
		{
			"dynamic-fields-multiple-entries",
			func(p *CSVParserConfig) {
				p.HeaderLabel = "Fields"
				p.FieldDelimiter = ","
			},
			[]entry.Entry{
				{
					Labels: map[string]string{
						"Fields": "name,age,height,number",
					},
					Body: "stanza dev,1,400,555-555-5555",
				},
				{
					Labels: map[string]string{
						"Fields": "x,y",
					},
					Body: "000100,2",
				},
				{
					Labels: map[string]string{
						"Fields": "a,b,c,d,e,f",
					},
					Body: "1,2,3,4,5,6",
				},
			},
			[]interface{}{
				map[string]interface{}{
					"name":   "stanza dev",
					"age":    "1",
					"height": "400",
					"number": "555-555-5555",
				},
				map[string]interface{}{
					"x": "000100",
					"y": "2",
				},
				map[string]interface{}{
					"a": "1",
					"b": "2",
					"c": "3",
					"d": "4",
					"e": "5",
					"f": "6",
				},
			},
			false,
			false,
		},
		{
			"dynamic-fields-tab",
			func(p *CSVParserConfig) {
				p.HeaderLabel = "columns"
				p.FieldDelimiter = ","
				p.HeaderDelimiter = "\t"
			},
			[]entry.Entry{
				{
					Labels: map[string]string{
						"columns": "name	age	height	number",
					},
					Body: "stanza dev,1,400,555-555-5555",
				},
			},
			[]interface{}{
				map[string]interface{}{
					"name":   "stanza dev",
					"age":    "1",
					"height": "400",
					"number": "555-555-5555",
				},
			},
			false,
			false,
		},
		{
			"dynamic-fields-label-missing",
			func(p *CSVParserConfig) {
				p.HeaderLabel = "Fields"
				p.FieldDelimiter = ","
			},
			[]entry.Entry{
				{
					Body: "stanza dev,1,400,555-555-5555",
				},
			},
			[]interface{}{
				map[string]interface{}{
					"name":   "stanza dev",
					"age":    "1",
					"height": "400",
					"number": "555-555-5555",
				},
			},
			false,
			true,
		},
		{
			"missing-header-field",
			func(p *CSVParserConfig) {
				p.FieldDelimiter = ","
			},
			[]entry.Entry{
				{
					Body: "stanza,1,400,555-555-5555",
				},
			},
			[]interface{}{
				map[string]interface{}{
					"name":   "stanza",
					"age":    "1",
					"height": "400",
					"number": "555-555-5555",
				},
			},
			true,
			false,
		},
		{
			"header-and-dynamic-header",
			func(p *CSVParserConfig) {
				p.Header = "name,age,height,number"
				p.HeaderDelimiter = "Fields"
				p.FieldDelimiter = ","
			},
			[]entry.Entry{
				{
					Body: "stanza,1,400,555-555-5555",
				},
			},
			[]interface{}{
				map[string]interface{}{
					"name":   "stanza",
					"age":    "1",
					"height": "400",
					"number": "555-555-5555",
				},
			},
			true,
			false,
		},
		{
			"mariadb-audit-log",
			func(p *CSVParserConfig) {
				p.Header = "timestamp,serverhost,username,host,connectionid,queryid,operation,database,object,retcode"
			},
			[]entry.Entry{
				{
					Body: "20210316 17:08:01,oiq-int-mysql,load,oiq-int-mysql.bluemedora.localnet,5,0,DISCONNECT,,,0",
				},
			},
			[]interface{}{
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
			false,
			false,
		},
		{
			"empty field",
			func(p *CSVParserConfig) {
				p.Header = "name,address,age,phone,position"
			},
			[]entry.Entry{
				{
					Body: "stanza,Evergreen,,555-5555,agent",
				},
			},
			[]interface{}{
				map[string]interface{}{
					"name":     "stanza",
					"address":  "Evergreen",
					"age":      "",
					"phone":    "555-5555",
					"position": "agent",
				},
			},
			false,
			false,
		},
		{
			"tab delimiter",
			func(p *CSVParserConfig) {
				p.Header = "name	address	age	phone	position"
				p.FieldDelimiter = "\t"
			},
			[]entry.Entry{
				{
					Body: "stanza	Evergreen	1	555-5555	agent",
				},
			},
			[]interface{}{
				map[string]interface{}{
					"name":     "stanza",
					"address":  "Evergreen",
					"age":      "1",
					"phone":    "555-5555",
					"position": "agent",
				},
			},
			false,
			false,
		},
		{
			"comma in quotes",
			func(p *CSVParserConfig) {
				p.Header = "name,address,age,phone,position"
			},
			[]entry.Entry{
				{
					Body: "stanza,\"Evergreen,49508\",1,555-5555,agent",
				},
			},
			[]interface{}{
				map[string]interface{}{
					"name":     "stanza",
					"address":  "Evergreen,49508",
					"age":      "1",
					"phone":    "555-5555",
					"position": "agent",
				},
			},
			false,
			false,
		},
		{
			"quotes in quotes",
			func(p *CSVParserConfig) {
				p.Header = "name,address,age,phone,position"
			},
			[]entry.Entry{
				{
					Body: "\"bob \"\"the man\"\"\",Evergreen,1,555-5555,agent",
				},
			},
			[]interface{}{
				map[string]interface{}{
					"name":     "bob \"the man\"",
					"address":  "Evergreen",
					"age":      "1",
					"phone":    "555-5555",
					"position": "agent",
				},
			},
			false,
			false,
		},
		{
			"header-delimiter",
			func(p *CSVParserConfig) {
				p.Header = "name+sev+msg"
				p.HeaderDelimiter = "+"
			},
			[]entry.Entry{
				{
					Body: "stanza,INFO,started agent",
				},
			},
			[]interface{}{
				map[string]interface{}{
					"name": "stanza",
					"sev":  "INFO",
					"msg":  "started agent",
				},
			},
			false,
			false,
		},
		{
			"tab-delimiter",
			func(p *CSVParserConfig) {
				p.Header = testHeader
				p.HeaderDelimiter = ","
				p.FieldDelimiter = "\t"
			},
			[]entry.Entry{
				{
					Body: "stanza\tINFO\tstarted agent",
				},
			},
			[]interface{}{
				map[string]interface{}{
					"name": "stanza",
					"sev":  "INFO",
					"msg":  "started agent",
				},
			},
			false,
			false,
		},
		{
			"missing-header-delimiter-in-header",
			func(p *CSVParserConfig) {
				p.Header = "name:age:height:number"
				p.FieldDelimiter = ","
			},
			[]entry.Entry{
				{
					Body: "stanza,1,400,555-555-5555",
				},
			},
			[]interface{}{
				map[string]interface{}{
					"name":   "stanza",
					"age":    "1",
					"height": "400",
					"number": "555-555-5555",
				},
			},
			true,
			false,
		},
		{
			"invalid-delimiter",
			func(p *CSVParserConfig) {
				// expect []rune of length 1
				p.Header = "name,,age,,height,,number"
				p.FieldDelimiter = ",,"
			},
			[]entry.Entry{
				{
					Body: "stanza,1,400,555-555-5555",
				},
			},
			[]interface{}{
				map[string]interface{}{
					"name":   "stanza",
					"age":    "1",
					"height": "400",
					"number": "555-555-5555",
				},
			},
			true,
			false,
		},
		{
			"parse-failure-num-fields-mismatch",
			func(p *CSVParserConfig) {
				p.Header = "name,age,height,number"
				p.FieldDelimiter = ","
			},
			[]entry.Entry{
				{
					Body: "1,400,555-555-5555",
				},
			},
			[]interface{}{
				map[string]interface{}{
					"name":   "stanza",
					"age":    "1",
					"height": "400",
					"number": "555-555-5555",
				},
			},
			false,
			true,
		},
		{
			"parse-failure-wrong-field-delimiter",
			func(p *CSVParserConfig) {
				p.Header = "name,age,height,number"
				p.FieldDelimiter = ","
			},
			[]entry.Entry{
				{
					Body: "stanza:1:400:555-555-5555",
				},
			},
			[]interface{}{
				map[string]interface{}{
					"name":   "stanza",
					"age":    "1",
					"height": "400",
					"number": "555-555-5555",
				},
			},
			false,
			true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			cfg := NewCSVParserConfig("test")
			cfg.OutputIDs = []string{"fake"}
			tc.configure(cfg)

			ops, err := cfg.Build(testutil.NewBuildContext(t))
			if tc.expectBuildErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			op := ops[0]

			fake := testutil.NewFakeOutput(t)
			op.SetOutputs([]operator.Operator{fake})

			for i, inputEntry := range tc.inputEntry {
				err = op.Process(context.Background(), &inputEntry)
				if tc.expectProcessErr {
					require.Error(t, err)
					return
				}
				require.NoError(t, err)

				fake.ExpectBody(t, tc.outputBody[i])
			}
		})
	}
}

func TestParserCSVMultipleBodys(t *testing.T) {
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
		entry.Body = "stanza,INFO,started agent\nstanza,DEBUG,started agent"
		err = op.Process(context.Background(), entry)
		// require.Nil(t, err, "Expected to parse a single csv body, got '2'")
		// require.Contains(t, err.Error(), "Expected to parse a single csv body, got '2'")
		require.NoError(t, err)

		fake.ExpectBody(t, map[string]interface{}{
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
		entry.Body = "{\"name\": \"stanza\"}"
		err = op.Process(context.Background(), entry)
		require.Error(t, err, "parse error on line 1, column 1: bare \" in non-quoted-field")
		fake.ExpectBody(t, "{\"name\": \"stanza\"}")
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
