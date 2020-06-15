package plugin

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/bluemedora/bplogagent/errors"
	"gopkg.in/yaml.v2"
)

// CustomConfig is the rendered config of a custom plugin.
type CustomConfig struct {
	Pipeline []Config
}

// CustomRegistry is a registry of custom plugin templates.
type CustomRegistry map[string]*template.Template

// Render will render a custom config using the params and plugin type.
func (r CustomRegistry) Render(pluginType string, params map[string]interface{}) (CustomConfig, error) {
	template, ok := r[pluginType]
	if !ok {
		return CustomConfig{}, errors.NewError(
			"custom plugin type does not exist",
			"ensure that all plugins are defined with a registered type",
			"plugin_type", pluginType,
		)
	}

	var writer bytes.Buffer
	if err := template.Execute(&writer, params); err != nil {
		return CustomConfig{}, errors.NewError(
			"failed to render template for custom plugin",
			"ensure that all parameters are valid for the custom plugin",
			"plugin_type", pluginType,
			"error_message", err.Error(),
		)
	}

	var config CustomConfig
	if err := yaml.Unmarshal(writer.Bytes(), &config); err != nil {
		return CustomConfig{}, errors.NewError(
			"failed to unmarshal custom template to custom config",
			"ensure that the custom template renders a valid pipeline",
			"plugin_type", pluginType,
			"rendered_config", writer.String(),
			"error_message", err.Error(),
		)
	}

	return config, nil
}

// IsDefined returns a boolean indicating if a custom plugin is defined and registered.
func (r CustomRegistry) IsDefined(pluginType string) bool {
	_, ok := r[pluginType]
	return ok
}

// LoadAll will load all custom plugin templates contained in a directory.
func (r CustomRegistry) LoadAll(dir string, pattern string) error {
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
		if err := r.Load(path); err != nil {
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
func (r CustomRegistry) Load(path string) error {
	fileName := filepath.Base(path)
	pluginType := strings.TrimSuffix(fileName, filepath.Ext(fileName))

	fileContents, err := ioutil.ReadFile(path)
	if err != nil {
		return fmt.Errorf("failed to read %s: %s", path, err)
	}

	return r.Add(pluginType, string(fileContents))
}

// Add will add a custom plugin to the registry.
func (r CustomRegistry) Add(pluginType string, contents string) error {
	if IsDefined(pluginType) {
		return fmt.Errorf("plugin type %s already exists as a builtin plugin", pluginType)
	}

	pluginTemplate, err := template.New(pluginType).Parse(contents)
	if err != nil {
		return fmt.Errorf("failed to parse %s as a custom template: %s", pluginType, err)
	}

	r[pluginType] = pluginTemplate
	return nil
}

// NewCustomRegistry creates a new custom plugin registry from a plugin directory.
func NewCustomRegistry(dir string) (CustomRegistry, error) {
	registry := CustomRegistry{}
	if err := registry.LoadAll(dir, "*.yaml"); err != nil {
		return registry, err
	}
	return registry, nil
}
