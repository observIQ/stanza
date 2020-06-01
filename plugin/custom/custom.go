package custom

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/bluemedora/bplogagent/errors"
	"github.com/bluemedora/bplogagent/plugin"
	"github.com/bluemedora/bplogagent/plugin/helper"
	yaml "gopkg.in/yaml.v2"
)

// registry is a registry of custom plugin templates.
var registry = make(map[string]*template.Template)

// Config is the rendered config of a custom plugin.
type Config struct {
	Pipeline []plugin.Config
}

// BuildConfig will build a custom config from a params map.
func BuildConfig(params map[string]interface{}, namespace string) (Config, error) {
	pluginID := getString(params, "id")
	pluginType := getString(params, "type")
	pluginOutput := getString(params, "output")

	input := helper.AddNamespace(pluginID, namespace)
	output := helper.AddNamespace(pluginOutput, namespace)
	templateParams := createTemplateParams(params, input, output)

	template, ok := registry[pluginType]
	if !ok {
		return Config{}, errors.NewError(
			"custom plugin type does not exist",
			"ensure that all plugins are defined with a registered type",
			"plugin_type", pluginType,
		)
	}

	var writer bytes.Buffer
	if err := template.Execute(&writer, templateParams); err != nil {
		return Config{}, errors.NewError(
			"failed to render template for custom plugin",
			"ensure that all parameters are valid for the custom plugin",
			"plugin_type", pluginType,
			"error_message", err.Error(),
		)
	}

	var config Config
	if err := yaml.Unmarshal(writer.Bytes(), &config); err != nil {
		return Config{}, errors.NewError(
			"failed to unmarshal custom template to custom config",
			"ensure that the custom template renders a valid pipeline",
			"plugin_type", pluginType,
			"error_message", err.Error(),
		)
	}

	for _, pluginConfig := range config.Pipeline {
		pluginConfig.SetNamespace(input, input, output)
	}

	return config, nil
}

// createTemplateParams will create the params used to render a custom plugin template.
func createTemplateParams(params map[string]interface{}, input string, output string) map[string]interface{} {
	templateParams := map[string]interface{}{}

	for key, value := range params {
		templateParams[key] = value
	}

	templateParams["input"] = input
	templateParams["output"] = output
	return templateParams
}

// getString retrieves a string from the params map.
func getString(params map[string]interface{}, key string) string {
	rawValue, ok := params[key]
	if !ok {
		return ""
	}

	stringValue, ok := rawValue.(string)
	if !ok {
		return ""
	}

	return stringValue
}

// IsDefined returns a boolean indicating if a custom plugin is defined and registered.
func IsDefined(pluginType string) bool {
	_, ok := registry[pluginType]
	return ok
}

// LoadAll will load all custom plugin templates contained in a directory.
func LoadAll(dir string, pattern string) error {
	glob := filepath.Join(dir, pattern)
	filePaths, err := filepath.Glob(glob)
	if err != nil {
		return errors.NewError(
			"failed to find custom plugins with glob pattern",
			"ensure that the plugin directory and file pattern are valid",
			"glob_pattern", glob,
		)
	}

	failures := make([]string, 0)
	for _, path := range filePaths {
		if err := Load(path); err != nil {
			failures = append(failures, err.Error())
		}
	}

	if len(failures) > 0 {
		return errors.NewError(
			"failed to load all custom plugins",
			"review the list of failures for more information",
			"failures", strings.Join(failures, ", "),
		)
	}

	return nil
}

// Load will load a custom plugin template from a file path.
func Load(path string) error {
	fileName := filepath.Base(path)
	pluginType := strings.TrimSuffix(fileName, filepath.Ext(fileName))

	fileContents, err := ioutil.ReadFile(path)
	if err != nil {
		return fmt.Errorf("failed to read %s: %s", path, err)
	}

	pluginTemplate, err := template.New(pluginType).Parse(string(fileContents))
	if err != nil {
		return fmt.Errorf("failed to parse %s as a template: %s", path, err)
	}

	registry[pluginType] = pluginTemplate
	return nil
}
