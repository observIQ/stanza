package builtin

import (
	"bytes"
	"context"
	"encoding/json"
	"strconv"

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

	Addresses  []string     `mapstructure:"addresses"   json:"addresses"             yaml:"addresses,flow"`
	Username   string       `mapstructure:"username"    json:"username"              yaml:"username"`
	Password   string       `mapstructure:"password"    json:"password"              yaml:"password"`
	CloudID    string       `mapstructure:"cloud_id"    json:"cloud_id"              yaml:"cloud_id"`
	APIKey     string       `mapstructure:"api_key"     json:"api_key"               yaml:"api_key"`
	IndexField *entry.Field `mapstructure:"index_field" json:"index_field,omitempty" yaml:"index_field,omitempty"`
	IDField    *entry.Field `mapstructure:"id_field"    json:"id_field,omitempty"    yaml:"id_field,omitempty"`
}

// Build will build an elasticsearch output plugin.
func (c ElasticOutputConfig) Build(context plugin.BuildContext) (plugin.Plugin, error) {
	basicPlugin, err := c.BasicPluginConfig.Build(context.Logger)
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
			"The elasticsearch client failed to initialize.",
			"Review the underlying error message to troubleshoot the issue.",
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
	indexField *entry.Field
	idField    *entry.Field
}

// Process will send entries to elasticsearch.
func (e *ElasticOutput) Process(ctx context.Context, entry *entry.Entry) error {
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
		Body:       bytes.NewReader(json),
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
