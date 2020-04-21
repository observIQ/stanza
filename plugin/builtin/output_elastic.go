package builtin

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/bluemedora/bplogagent/entry"
	"github.com/bluemedora/bplogagent/errors"
	"github.com/bluemedora/bplogagent/plugin"
	"github.com/bluemedora/bplogagent/plugin/helper"
	"github.com/elastic/go-elasticsearch/v8"
	"github.com/elastic/go-elasticsearch/v8/esapi"
	"github.com/hashicorp/go-uuid"
)

func init() {
	plugin.Register("elastic_output", &ElasticOutputConfig{})
}

// ElasticOutputConfig is the configuration of an elasticsearch output plugin.
type ElasticOutputConfig struct {
	helper.BasicPluginConfig `mapstructure:",squash" yaml:",inline"`
	elasticsearch.Config     `mapstructure:",squash" yaml:",inline"`
	IndexField               entry.FieldSelector `mapstructure:"index_field" yaml:"index_field"`
	IDField                  entry.FieldSelector `mapstructure:"id_field" yaml:"id_field"`
}

// Build will build a logger output plugin.
func (c ElasticOutputConfig) Build(context plugin.BuildContext) (plugin.Plugin, error) {
	basicPlugin, err := c.BasicPluginConfig.Build(context.Logger)
	if err != nil {
		return nil, err
	}

	client, err := elasticsearch.NewClient(c.Config)
	if err != nil {
		return nil, errors.NewError(
			"The elasticsearch client failed to initialize.",
			"Use the underlying error message to troubleshoot the issue.",
			"underlying_error", err.Error(),
		)
	}

	elasticOutput := &ElasticOutput{
		BasicPlugin: basicPlugin,
		client:      client,
		indexField:  c.IndexField,
		idField:     c.IDField,
	}

	return elasticOutput, nil
}

// ElasticOutput is a plugin that sends entries to elasticsearch.
type ElasticOutput struct {
	helper.BasicPlugin
	helper.BasicLifecycle
	helper.BasicOutput

	client     *elasticsearch.Client
	indexField entry.FieldSelector
	idField    entry.FieldSelector
}

// Process will send entries to elasticsearch.
func (e *ElasticOutput) Process(entry *entry.Entry) error {
	json, err := json.Marshal(entry)
	if err != nil {
		return err
	}

	index, err := e.FindIndex(entry)
	if err != nil {
		return err
	}

	id, err := e.FindID(entry)
	if err != nil {
		return err
	}

	request := esapi.IndexRequest{
		Index:      index,
		DocumentID: id,
		Body:       strings.NewReader(string(json)),
	}

	res, err := request.Do(context.Background(), e.client)
	if err != nil {
		return errors.NewError(
			"Client failed to submit entry to elasticsearch.",
			"Use the underlying error to troubleshoot the problem",
			"underlying_error", err.Error(),
		)
	}

	defer res.Body.Close()
	return nil
}

// FindIndex will find an index that will represent an entry in elasticsearch.
func (e *ElasticOutput) FindIndex(entry *entry.Entry) (string, error) {
	if e.indexField == nil {
		return "default", nil
	}

	value, ok := e.indexField.Get(entry.Record)
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

	value, ok := e.idField.Get(entry.Record)
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
