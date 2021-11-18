package uri

import (
	"context"
	"fmt"
	"net/url"
	"strings"

	"github.com/observiq/stanza/v2/entry"
	"github.com/observiq/stanza/v2/operator"
	"github.com/observiq/stanza/v2/operator/helper"
)

func init() {
	operator.Register("uri_parser", func() operator.Builder { return NewURIParserConfig("") })
}

// NewURIParserConfig creates a new uri parser config with default values.
func NewURIParserConfig(operatorID string) *URIParserConfig {
	return &URIParserConfig{
		ParserConfig: helper.NewParserConfig(operatorID, "uri_parser"),
	}
}

// URIParserConfig is the configuration of a uri parser operator.
type URIParserConfig struct {
	helper.ParserConfig `yaml:",inline"`
}

// Build will build a uri parser operator.
func (c URIParserConfig) Build(context operator.BuildContext) ([]operator.Operator, error) {
	parserOperator, err := c.ParserConfig.Build(context)
	if err != nil {
		return nil, err
	}

	uriParser := &URIParser{
		ParserOperator: parserOperator,
	}

	return []operator.Operator{uriParser}, nil
}

// URIParser is an operator that parses a uri.
type URIParser struct {
	helper.ParserOperator
}

// Process will parse an entry.
func (u *URIParser) Process(ctx context.Context, entry *entry.Entry) error {
	return u.ParserOperator.ProcessWith(ctx, entry, u.parse)
}

// parse will parse a uri from a field and attach it to an entry.
func (u *URIParser) parse(value interface{}) (interface{}, error) {
	switch m := value.(type) {
	case string:
		return parseURI(m)
	case []byte:
		return parseURI(string(m))
	default:
		return nil, fmt.Errorf("type '%T' cannot be parsed as URI", value)
	}
}

// parseURI takes an absolute or relative uri and returns the parsed values.
func parseURI(value string) (map[string]interface{}, error) {
	m := make(map[string]interface{})

	if strings.HasPrefix(value, "?") {
		// remove the query string '?' prefix before parsing
		v, err := url.ParseQuery(value[1:])
		if err != nil {
			return nil, err
		}
		return queryToMap(v, m), nil
	}

	x, err := url.ParseRequestURI(value)
	if err != nil {
		return nil, err
	}
	return urlToMap(x, m), nil
}

// urlToMap converts a url.URL to a map, excludes any values that are not set.
func urlToMap(p *url.URL, m map[string]interface{}) map[string]interface{} {
	scheme := p.Scheme
	if scheme != "" {
		m["scheme"] = scheme
	}

	user := p.User.Username()
	if user != "" {
		m["user"] = user
	}

	host := p.Hostname()
	if host != "" {
		m["host"] = host
	}

	port := p.Port()
	if port != "" {
		m["port"] = port
	}

	path := p.EscapedPath()
	if path != "" {
		m["path"] = path
	}

	return queryToMap(p.Query(), m)
}

// queryToMap converts a query string url.Values to a map.
func queryToMap(query url.Values, m map[string]interface{}) map[string]interface{} {
	// no-op if query is empty, do not create the key m["query"]
	if len(query) <= 0 {
		return m
	}

	/* 'parameter' will represent url.Values
	map[string]interface{}{
		"parameter-a": []interface{}{
			"a",
			"b",
		},
		"parameter-b": []interface{}{
			"x",
			"y",
		},
	}
	*/
	parameters := map[string]interface{}{}
	for param, values := range query {
		parameters[param] = queryParamValuesToMap(values)
	}
	m["query"] = parameters
	return m
}

// queryParamValuesToMap takes query string parameter values and
// returns an []interface populated with the values
func queryParamValuesToMap(values []string) []interface{} {
	v := make([]interface{}, len(values))
	for i, value := range values {
		v[i] = value
	}
	return v
}
