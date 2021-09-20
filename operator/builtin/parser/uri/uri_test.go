package uri

import (
	"net/url"
	"testing"

	"github.com/observiq/stanza/testutil"

	"github.com/stretchr/testify/require"
)

func newTestParser(t *testing.T) *URIParser {
	cfg := NewURIParserConfig("test")
	ops, err := cfg.Build(testutil.NewBuildContext(t))
	require.NoError(t, err)
	op := ops[0]
	return op.(*URIParser)
}

func TestURIParserBuildFailure(t *testing.T) {
	cfg := NewURIParserConfig("test")
	cfg.OnError = "invalid_on_error"
	_, err := cfg.Build(testutil.NewBuildContext(t))
	require.Error(t, err)
	require.Contains(t, err.Error(), "invalid `on_error` field")
}

func TestURIParserStringFailure(t *testing.T) {
	parser := newTestParser(t)
	_, err := parser.parse("invalid")
	require.Error(t, err)
	require.Contains(t, err.Error(), "parse \"invalid\": invalid URI for request")
}

func TestURIParserByteFailure(t *testing.T) {
	parser := newTestParser(t)
	_, err := parser.parse([]byte("invalid"))
	require.Error(t, err)
	require.Contains(t, err.Error(), "parse \"invalid\": invalid URI for request")
}

func TestRegexParserInvalidType(t *testing.T) {
	parser := newTestParser(t)
	_, err := parser.parse([]int{})
	require.Error(t, err)
	require.Contains(t, err.Error(), "type '[]int' cannot be parsed as URI")
}

func TestURIParserParse(t *testing.T) {
	cases := []struct {
		name         string
		inputBody    interface{}
		expectedBody map[string]interface{}
		expectErr    bool
	}{
		{
			"string",
			"http://www.google.com/app?env=prod",
			map[string]interface{}{
				"scheme": "http",
				"host":   "www.google.com",
				"path":   "/app",
				"query": map[string]interface{}{
					"env": []interface{}{
						"prod",
					},
				},
			},
			false,
		},
		{
			"byte",
			[]byte("http://google.com/app?env=prod"),
			map[string]interface{}{
				"scheme": "http",
				"host":   "google.com",
				"path":   "/app",
				"query": map[string]interface{}{
					"env": []interface{}{
						"prod",
					},
				},
			},
			false,
		},
		{
			"byte",
			[]int{},
			map[string]interface{}{},
			true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			parser := URIParser{}
			x, err := parser.parse(tc.inputBody)
			if tc.expectErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tc.expectedBody, x)
		})
	}
}

// Test all usecases: absolute uri, relative uri, query string
func TestParseURI(t *testing.T) {
	cases := []struct {
		name         string
		inputBody    string
		expectedBody map[string]interface{}
		expectErr    bool
	}{
		{
			"scheme-http",
			"http://",
			map[string]interface{}{
				"scheme": "http",
			},
			false,
		},
		{
			"scheme-user",
			"http://myuser:mypass@",
			map[string]interface{}{
				"scheme": "http",
				"user":   "myuser",
			},
			false,
		},
		{
			"scheme-host",
			"http://golang.com",
			map[string]interface{}{
				"scheme": "http",
				"host":   "golang.com",
			},
			false,
		},
		{
			"scheme-host-root",
			"http://golang.com/",
			map[string]interface{}{
				"scheme": "http",
				"host":   "golang.com",
				"path":   "/",
			},
			false,
		},
		{
			"scheme-host-minimal",
			"http://golang",
			map[string]interface{}{
				"scheme": "http",
				"host":   "golang",
			},
			false,
		},
		{
			"host-missing-scheme",
			"golang.org",
			map[string]interface{}{},
			true,
		},
		{
			"sheme-port",
			"http://:8080",
			map[string]interface{}{
				"scheme": "http",
				"port":   "8080",
			},
			false,
		},
		{
			"port-missing-scheme",
			":8080",
			map[string]interface{}{},
			true,
		},
		{
			"path",
			"/docs",
			map[string]interface{}{
				"path": "/docs",
			},
			false,
		},
		{
			"path-advanced",
			`/x/y%2Fz`,
			map[string]interface{}{
				"path": `/x/y%2Fz`,
			},
			false,
		},
		{
			"path-root",
			"/",
			map[string]interface{}{
				"path": "/",
			},
			false,
		},
		{
			"path-query",
			"/v1/app?user=golang",
			map[string]interface{}{
				"path": "/v1/app",
				"query": map[string]interface{}{
					"user": []interface{}{
						"golang",
					},
				},
			},
			false,
		},
		{
			"scheme-path",
			"http:///v1/app",
			map[string]interface{}{
				"scheme": "http",
				"path":   "/v1/app",
			},
			false,
		},
		{
			"scheme-host-query",
			"https://app.com?token=0000&env=prod&env=stage",
			map[string]interface{}{
				"scheme": "https",
				"host":   "app.com",
				"query": map[string]interface{}{
					"token": []interface{}{
						"0000",
					},
					"env": []interface{}{
						"prod",
						"stage",
					},
				},
			},
			false,
		},
		{
			"minimal",
			"http://golang.org",
			map[string]interface{}{
				"scheme": "http",
				"host":   "golang.org",
			},
			false,
		},
		{
			"advanced",
			"https://go:password@golang.org:8443/v2/app?env=stage&token=456&index=105838&env=prod",
			map[string]interface{}{
				"scheme": "https",
				"user":   "go",
				"host":   "golang.org",
				"port":   "8443",
				"path":   "/v2/app",
				"query": map[string]interface{}{
					"token": []interface{}{
						"456",
					},
					"index": []interface{}{
						"105838",
					},
					"env": []interface{}{
						"stage",
						"prod",
					},
				},
			},
			false,
		},
		{
			"magnet",
			"magnet:?xt=urn:sha1:HNCKHTQCWBTRNJIV4WNAE52SJUQCZO6C",
			map[string]interface{}{
				"scheme": "magnet",
				"query": map[string]interface{}{
					"xt": []interface{}{
						"urn:sha1:HNCKHTQCWBTRNJIV4WNAE52SJUQCZO6C",
					},
				},
			},
			false,
		},
		{
			"sftp",
			"sftp://ftp.com//home/name/employee.csv",
			map[string]interface{}{
				"scheme": "sftp",
				"host":   "ftp.com",
				"path":   "//home/name/employee.csv",
			},
			false,
		},
		{
			"missing-schema",
			"golang.org/app",
			map[string]interface{}{},
			true,
		},
		{
			"query-advanced",
			"?token=0000&env=prod&env=stage&task=update&task=new&action=update",
			map[string]interface{}{
				"query": map[string]interface{}{
					"token": []interface{}{
						"0000",
					},
					"env": []interface{}{
						"prod",
						"stage",
					},
					"task": []interface{}{
						"update",
						"new",
					},
					"action": []interface{}{
						"update",
					},
				},
			},
			false,
		},
		{
			"query",
			"?token=0000",
			map[string]interface{}{
				"query": map[string]interface{}{
					"token": []interface{}{
						"0000",
					},
				},
			},
			false,
		},
		{
			"query-empty",
			"?",
			map[string]interface{}{},
			false,
		},
		{
			"query-empty-key",
			"?user=",
			map[string]interface{}{
				"query": map[string]interface{}{
					"user": []interface{}{
						"", // no value
					},
				},
			},
			false,
		},
		// Query string without a ? prefix is treated as a URI, therefor
		// an error will be returned by url.Parse("user=dev")
		{
			"query-no-?-prefix",
			"user=dev",
			map[string]interface{}{},
			true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			x, err := parseURI(tc.inputBody)
			if tc.expectErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tc.expectedBody, x)
		})
	}
}

func TestBuildParserURL(t *testing.T) {
	newBasicURIParser := func() *URIParserConfig {
		cfg := NewURIParserConfig("test")
		cfg.OutputIDs = []string{"test"}
		return cfg
	}

	t.Run("BasicConfig", func(t *testing.T) {
		c := newBasicURIParser()
		_, err := c.Build(testutil.NewBuildContext(t))
		require.NoError(t, err)
	})
}

func TestURLToMap(t *testing.T) {
	cases := []struct {
		name         string
		inputBody    url.URL
		expectedBody map[string]interface{}
	}{
		{
			"absolute-uri",
			url.URL{
				Scheme:   "https",
				Host:     "google.com:8443",
				Path:     "/app",
				RawQuery: "stage=prod&stage=dev",
			},
			map[string]interface{}{
				"scheme": "https",
				"host":   "google.com",
				"port":   "8443",
				"path":   "/app",
				"query": map[string]interface{}{
					"stage": []interface{}{
						"prod",
						"dev",
					},
				},
			},
		},
		{
			"absolute-uri-simple",
			url.URL{
				Scheme: "http",
				Host:   "google.com",
			},
			map[string]interface{}{
				"scheme": "http",
				"host":   "google.com",
			},
		},
		{
			"path",
			url.URL{
				Path:     "/app",
				RawQuery: "stage=prod&stage=dev",
			},
			map[string]interface{}{
				"path": "/app",
				"query": map[string]interface{}{
					"stage": []interface{}{
						"prod",
						"dev",
					},
				},
			},
		},
		{
			"path-simple",
			url.URL{
				Path: "/app",
			},
			map[string]interface{}{
				"path": "/app",
			},
		},
		{
			"query",
			url.URL{
				RawQuery: "stage=prod&stage=dev",
			},
			map[string]interface{}{
				"query": map[string]interface{}{
					"stage": []interface{}{
						"prod",
						"dev",
					},
				},
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			m := make(map[string]interface{})
			require.Equal(t, tc.expectedBody, urlToMap(&tc.inputBody, m))
		})
	}
}

func TestQueryToMap(t *testing.T) {
	cases := []struct {
		name         string
		inputBody    url.Values
		expectedBody map[string]interface{}
	}{
		{
			"query",
			url.Values{
				"stage": []string{
					"prod",
					"dev",
				},
			},
			map[string]interface{}{
				"query": map[string]interface{}{
					"stage": []interface{}{
						"prod",
						"dev",
					},
				},
			},
		},
		{
			"empty",
			url.Values{},
			map[string]interface{}{},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			m := make(map[string]interface{})
			require.Equal(t, tc.expectedBody, queryToMap(tc.inputBody, m))
		})
	}
}

func TestQueryParamValuesToMap(t *testing.T) {
	cases := []struct {
		name         string
		inputBody    []string
		expectedBody []interface{}
	}{
		{
			"simple",
			[]string{
				"prod",
				"dev",
			},
			[]interface{}{
				"prod",
				"dev",
			},
		},
		{
			"empty",
			[]string{},
			[]interface{}{},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			require.Equal(t, tc.expectedBody, queryParamValuesToMap(tc.inputBody))
		})
	}
}

func BenchmarkURIParserParse(b *testing.B) {
	v := "https://dev:password@www.golang.org:8443/v1/app/stage?token=d9e28b1d-2c7b-4853-be6a-d94f34a5d4ab&env=prod&env=stage&token=c6fa29f9-a31b-4584-b98d-aa8473b0e18d&region=us-east1b&mode=fast"
	parser := URIParser{}
	for n := 0; n < b.N; n++ {
		if _, err := parser.parse(v); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkURLToMap(b *testing.B) {
	m := make(map[string]interface{})
	v := "https://dev:password@www.golang.org:8443/v1/app/stage?token=d9e28b1d-2c7b-4853-be6a-d94f34a5d4ab&env=prod&env=stage&token=c6fa29f9-a31b-4584-b98d-aa8473b0e18d&region=us-east1b&mode=fast"
	u, err := url.ParseRequestURI(v)
	if err != nil {
		b.Fatal(err)
	}
	for n := 0; n < b.N; n++ {
		urlToMap(u, m)
	}
}

func BenchmarkQueryToMap(b *testing.B) {
	m := make(map[string]interface{})
	v := "?token=d9e28b1d-2c7b-4853-be6a-d94f34a5d4ab&env=prod&env=stage&token=c6fa29f9-a31b-4584-b98d-aa8473b0e18d&region=us-east1b&mode=fast"
	u, err := url.ParseQuery(v)
	if err != nil {
		b.Fatal(err)
	}
	for n := 0; n < b.N; n++ {
		queryToMap(u, m)
	}
}

func BenchmarkQueryParamValuesToMap(b *testing.B) {
	v := []string{
		"d9e28b1d-2c7b-4853-be6a-d94f34a5d4ab",
		"c6fa29f9-a31b-4584-b98d-aa8473b0e18",
	}
	for n := 0; n < b.N; n++ {
		queryParamValuesToMap(v)
	}
}
