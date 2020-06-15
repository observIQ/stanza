package output

import (
	"bytes"
	"context"
	"encoding/json"
	"strconv"

	"github.com/bluemedora/bplogagent/entry"
	"github.com/bluemedora/bplogagent/errors"
	"github.com/bluemedora/bplogagent/plugin"
	"github.com/bluemedora/bplogagent/plugin/buffer"
	"github.com/bluemedora/bplogagent/plugin/helper"
	elasticsearch "github.com/elastic/go-elasticsearch/v7"
	"github.com/elastic/go-elasticsearch/v7/esapi"
	uuid "github.com/hashicorp/go-uuid"
	"go.uber.org/zap"
)

func init() {
	plugin.Register("elastic_output", &ElasticOutputConfig{})
}

// ElasticOutputConfig is the configuration of an elasticsearch output plugin.
type ElasticOutputConfig struct {
	helper.OutputConfig `yaml:",inline"`
	buffer.BufferConfig `json:"buffer" yaml:"buffer"`

	Addresses  []string     `json:"addresses"             yaml:"addresses,flow"`
	Username   string       `json:"username"              yaml:"username"`
	Password   string       `json:"password"              yaml:"password"`
	CloudID    string       `json:"cloud_id"              yaml:"cloud_id"`
	APIKey     string       `json:"api_key"               yaml:"api_key"`
	IndexField *entry.Field `json:"index_field,omitempty" yaml:"index_field,omitempty"`
	IDField    *entry.Field `json:"id_field,omitempty"    yaml:"id_field,omitempty"`
}

// Build will build an elasticsearch output plugin.
func (c ElasticOutputConfig) Build(context plugin.BuildContext) (plugin.Plugin, error) {
	outputPlugin, err := c.OutputConfig.Build(context)
	if err != nil {
		return nil, err
	}

	cfg := elasticsearch.Config{
		Addresses: c.Addresses,
		Username:  c.Username,
		Password:  c.Password,
		CloudID:   c.CloudID,
		APIKey:    c.APIKey,
	}

	client, err := elasticsearch.NewClient(cfg)
	if err != nil {
		return nil, errors.NewError(
			"The Elasticsearch client failed to initialize.",
			"Review the underlying error message to troubleshoot the issue.",
			"underlying_error", err.Error(),
		)
	}

	buffer, err := c.BufferConfig.Build()
	if err != nil {
		return nil, err
	}

	elasticOutput := &ElasticOutput{
		OutputPlugin: outputPlugin,
		Buffer:       buffer,
		client:       client,
		indexField:   c.IndexField,
		idField:      c.IDField,
	}

	buffer.SetHandler(elasticOutput)

	return elasticOutput, nil
}

// ElasticOutput is a plugin that sends entries to elasticsearch.
type ElasticOutput struct {
	helper.OutputPlugin
	buffer.Buffer

	client     *elasticsearch.Client
	indexField *entry.Field
	idField    *entry.Field
}

// Process will send entries to elasticsearch.
func (e *ElasticOutput) ProcessMulti(ctx context.Context, entries []*entry.Entry) error {
	type indexDirective struct {
		Index struct {
			Index string `json:"_index"`
			ID    string `json:"_id"`
		} `json:"index"`
	}

	// The bulk API expects newline-delimited json strings, with an operation directive
	// immediately followed by the document.
	// https://www.elastic.co/guide/en/elasticsearch/reference/master/docs-bulk.html
	var buffer bytes.Buffer
	var err error
	for _, entry := range entries {

		directive := indexDirective{}
		directive.Index.Index, err = e.FindIndex(entry)
		if err != nil {
			e.Warnw("Failed to find index", zap.Any("error", err))
			continue
		}

		directive.Index.ID, err = e.FindID(entry)
		if err != nil {
			e.Warnw("Failed to find id", zap.Any("error", err))
			continue
		}

		directiveJSON, err := json.Marshal(directive)
		if err != nil {
			e.Warnw("Failed to marshal directive JSON", zap.Any("error", err))
			continue
		}

		entryJSON, err := json.Marshal(entry)
		if err != nil {
			e.Warnw("Failed to marshal entry JSON", zap.Any("error", err))
			continue
		}

		buffer.Write(directiveJSON)
		buffer.Write([]byte("\n"))
		buffer.Write(entryJSON)
		buffer.Write([]byte("\n"))
	}

	request := esapi.BulkRequest{
		Body: bytes.NewReader(buffer.Bytes()),
	}

	res, err := request.Do(ctx, e.client)
	if err != nil {
		return errors.NewError(
			"Client failed to submit request to elasticsearch.",
			"Review the underlying error message to troubleshoot the issue",
			"underlying_error", err.Error(),
		)
	}

	defer res.Body.Close()

	if res.IsError() {
		return errors.NewError(
			"Request to elasticsearch returned a failure code.",
			"Review status and status code for further details.",
			"status_code", strconv.Itoa(res.StatusCode),
			"status", res.Status(),
		)
	}

	return nil
}

// FindIndex will find an index that will represent an entry in elasticsearch.
func (e *ElasticOutput) FindIndex(entry *entry.Entry) (string, error) {
	if e.indexField == nil {
		return "default", nil
	}

	value, ok := e.indexField.Get(entry)
	if !ok {
		return "", errors.NewError(
			"Failed to extract index from record.",
			"Ensure that all records contain the assigned index field.",
		)
	}

	strValue, ok := value.(string)
	if !ok {
		return "", errors.NewError(
			"Extracted index is not a string.",
			"Ensure that the index field contains a string value.",
		)
	}

	return strValue, nil
}

// FindID will find the id that will represent an entry in elasticsearch.
func (e *ElasticOutput) FindID(entry *entry.Entry) (string, error) {
	if e.idField == nil {
		return uuid.GenerateUUID()
	}

	value, ok := e.idField.Get(entry)
	if !ok {
		return "", errors.NewError(
			"Failed to extract id from record.",
			"Ensure that all records contain the assigned id field.",
		)
	}

	strValue, ok := value.(string)
	if !ok {
		return "", errors.NewError(
			"Extracted id is not a string.",
			"Ensure that the id field contains a string value.",
		)
	}

	return strValue, nil
}
