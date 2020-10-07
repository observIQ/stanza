package plugin

import (
	"bytes"
	"io/ioutil"
	"path/filepath"
	"regexp"
	"strings"
	"text/template"

	"github.com/observiq/stanza/errors"
	"github.com/observiq/stanza/operator"
	yaml "gopkg.in/yaml.v2"
)

// Plugin is the rendered result of a plugin template.
type Plugin struct {
	ID          string
	Version     string
	Title       string
	Type        string
	Description string
	Parameters  map[string]Parameter
	Template    *template.Template
}

// NewBuilder creates a new, empty config that can build into an operator
func (p *Plugin) NewBuilder() operator.MultiBuilder {
	return &Config{
		Plugin: p,
	}
}

// Render will render a plugin's template with the given parameters
func (p *Plugin) Render(params map[string]interface{}) ([]byte, error) {
	if err := p.Validate(params); err != nil {
		return nil, err
	}

	var writer bytes.Buffer
	if err := p.Template.Execute(&writer, params); err != nil {
		return nil, errors.NewError(
			"failed to render template for plugin",
			"ensure that all parameters are valid for the plugin",
			"plugin_type", p.ID,
			"error_message", err.Error(),
		)
	}

	return writer.Bytes(), nil
}

// Validate checks the provided params against the parameter definitions to ensure they are valid
func (p *Plugin) Validate(params map[string]interface{}) error {
	for name, param := range p.Parameters {
		value, ok := params[name]
		if !ok && !param.Required {
			continue
		}

		if !ok && param.Required {
			return errors.NewError(
				"missing required parameter for plugin",
				"ensure that the parameter is defined for the plugin",
				"plugin_type", p.ID,
				"plugin_parameter", name,
			)
		}

		if err := param.validateValue(value); err != nil {
			return errors.NewError(
				"plugin parameter failed validation",
				"review the underlying error message for details",
				"plugin_type", p.ID,
				"plugin_parameter", name,
				"error_message", err.Error(),
			)
		}
	}

	return nil
}

// UnmarshalText unmarshals a plugin from a text file
func (p *Plugin) UnmarshalText(text []byte) error {
	metadataBytes, templateBytes, err := splitPluginFile(text)
	if err != nil {
		return err
	}

	if err := yaml.Unmarshal(metadataBytes, p); err != nil {
		return err
	}

	p.Template, err = template.New(p.Title).
		Funcs(pluginFuncs()).
		Parse(string(templateBytes))
	return err
}

func splitPluginFile(text []byte) (metadata, template []byte, err error) {
	// Split the file into the metadata and the template by finding the pipeline,
	// then navigating backwards until we find a non-commented, non-empty line
	var metadataBuf bytes.Buffer
	var templateBuf bytes.Buffer

	lines := bytes.Split(text, []byte("\n"))
	if len(lines) != 0 && len(lines[len(lines)-1]) == 0 {
		// Delete empty trailing line
		lines = lines[:len(lines)-1]
	}

	// Find the index of the pipeline line
	pipelineRegex := regexp.MustCompile(`^pipeline:`)
	pipelineIndex := -1
	for i, line := range lines {
		if pipelineRegex.Match(line) {
			pipelineIndex = i
			break
		}
	}

	if pipelineIndex == -1 {
		return nil, nil, errors.NewError(
			"missing the pipeline block in plugin template",
			"ensure that the plugin file contains a pipeline",
		)
	}

	// Iterate backwards from the pipeline start to find the first non-commented, non-empty line
	emptyRegexp := regexp.MustCompile(`^\s*$`)
	commentedRegexp := regexp.MustCompile(`^\s*#`)
	templateStartIndex := pipelineIndex
	for i := templateStartIndex - 1; i >= 0; i-- {
		line := lines[i]
		if emptyRegexp.Match(line) || commentedRegexp.Match(line) {
			templateStartIndex = i
			continue
		}
		break
	}

	for _, line := range lines[:templateStartIndex] {
		metadataBuf.Write(line)
		metadataBuf.WriteByte('\n')
	}

	for _, line := range lines[templateStartIndex:] {
		templateBuf.Write(line)
		templateBuf.WriteByte('\n')
	}

	return metadataBuf.Bytes(), templateBuf.Bytes(), nil
}

// NewPluginFromFile builds a new plugin from a file
func NewPluginFromFile(path string) (*Plugin, error) {
	contents, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}

	id := strings.Split(filepath.Base(path), ".")[0]
	return NewPlugin(id, contents)
}

// NewPlugin builds a new plugin from an ID and file contents
func NewPlugin(id string, contents []byte) (*Plugin, error) {
	p := &Plugin{}
	if err := p.UnmarshalText(contents); err != nil {
		return nil, err
	}
	p.ID = id

	// Validate the parameter definitions
	for name, param := range p.Parameters {
		if err := param.validateDefinition(); err != nil {
			return nil, errors.NewError(
				"invalid parameter found in plugin",
				"ensure that all parameters are valid for the plugin",
				"plugin_type", p.ID,
				"plugin_parameter", name,
				"error_message", err.Error(),
			)
		}
	}

	return p, nil
}

// RegisterPlugins adds every plugin in a directory to the global plugin registry
func RegisterPlugins(pluginDir string, registry *operator.Registry) error {
	glob := filepath.Join(pluginDir, "*.yaml")
	filePaths, err := filepath.Glob(glob)
	if err != nil {
		return errors.NewError(
			"failed to find plugins with glob pattern",
			"ensure that the plugin directory and file pattern are valid",
			"glob_pattern", glob,
		)
	}

	for _, path := range filePaths {
		plugin, err := NewPluginFromFile(path)
		if err != nil {
			return errors.Wrap(err, "parse plugin file")
		}
		registry.RegisterPlugin(plugin.ID, plugin.NewBuilder)
	}

	return nil
}

// pluginFuncs returns a map of custom plugin functions used for templating.
func pluginFuncs() template.FuncMap {
	funcs := make(map[string]interface{})
	funcs["default"] = defaultPluginFunc
	return funcs
}

// defaultPluginFunc is a plugin function that returns a default value if the supplied value is nil.
func defaultPluginFunc(def interface{}, val interface{}) interface{} {
	if val == nil {
		return def
	}
	return val
}
