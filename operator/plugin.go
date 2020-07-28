package operator

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/observiq/carbon/errors"
	yaml "gopkg.in/yaml.v2"
)

// PluginConfig is the rendered config of a plugin.
type PluginConfig struct {
	Version     string
	Title       string
	Description string
	Parameters  map[string]PluginParameter
	Pipeline    []Config
}

// PluginRegistry is a registry of plugin templates.
type PluginRegistry map[string]*template.Template

// Render will render a plugin config using the params and plugin type.
func (r PluginRegistry) Render(pluginType string, params map[string]interface{}) (PluginConfig, error) {
	template, ok := r[pluginType]
	if !ok {
		return PluginConfig{}, errors.NewError(
			"plugin type does not exist",
			"ensure that all plugins are defined with a registered type",
			"plugin_type", pluginType,
		)
	}

	var writer bytes.Buffer
	if err := template.Execute(&writer, params); err != nil {
		return PluginConfig{}, errors.NewError(
			"failed to render template for plugin",
			"ensure that all parameters are valid for the plugin",
			"plugin_type", pluginType,
			"error_message", err.Error(),
		)
	}

	var config PluginConfig
	if err := yaml.UnmarshalStrict(writer.Bytes(), &config); err != nil {
		return PluginConfig{}, errors.NewError(
			"failed to unmarshal plugin template to plugin config",
			"ensure that the plugin template renders a valid pipeline",
			"plugin_type", pluginType,
			"rendered_config", writer.String(),
			"error_message", err.Error(),
		)
	}

	for name, param := range config.Parameters {
		if err := param.validate(); err != nil {
			return PluginConfig{}, errors.NewError(
				"invalid parameter found in plugin config",
				"ensure that all parameters are valid for the plugin",
				"plugin_type", pluginType,
				"plugin_parameter", name,
				"rendered_config", writer.String(),
				"error_message", err.Error(),
			)
		}
	}

	return config, nil
}

// IsDefined returns a boolean indicating if a plugin is defined and registered.
func (r PluginRegistry) IsDefined(pluginType string) bool {
	_, ok := r[pluginType]
	return ok
}

// LoadAll will load all plugin templates contained in a directory.
func (r PluginRegistry) LoadAll(dir string, pattern string) error {
	glob := filepath.Join(dir, pattern)
	filePaths, err := filepath.Glob(glob)
	if err != nil {
		return errors.NewError(
			"failed to find plugins with glob pattern",
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
			"failed to load all plugins",
			"review the list of failures for more information",
			"failures", strings.Join(failures, ", "),
		)
	}

	return nil
}

// Load will load a plugin template from a file path.
func (r PluginRegistry) Load(path string) error {
	fileName := filepath.Base(path)
	pluginType := strings.TrimSuffix(fileName, filepath.Ext(fileName))

	fileContents, err := ioutil.ReadFile(path)
	if err != nil {
		return fmt.Errorf("failed to read %s: %s", path, err)
	}

	return r.Add(pluginType, string(fileContents))
}

// Add will add a plugin to the registry.
func (r PluginRegistry) Add(pluginType string, contents string) error {
	if IsDefined(pluginType) {
		return fmt.Errorf("plugin type %s already exists as a builtin plugin", pluginType)
	}

	pluginTemplate, err := template.New(pluginType).Parse(contents)
	if err != nil {
		return fmt.Errorf("failed to parse %s as a plugin template: %s", pluginType, err)
	}

	r[pluginType] = pluginTemplate
	return nil
}

// NewPluginRegistry creates a new plugin registry from a plugin directory.
func NewPluginRegistry(dir string) (PluginRegistry, error) {
	registry := PluginRegistry{}
	if err := registry.LoadAll(dir, "*.yaml"); err != nil {
		return registry, err
	}
	return registry, nil
}
