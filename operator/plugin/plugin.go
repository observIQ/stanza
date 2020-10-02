package plugin

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
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
	Description string
	Parameters  map[string]Parameter
	Template    *template.Template
}

// NewBuilder creates a new, empty config that can build into an operator
func (p *Plugin) NewBuilder() operator.MultiBuilder {
	return &Config{
		plugin: p,
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

func (p *Plugin) UnmarshalText(text []byte) error {
	metadataBytes, templateBytes, err := splitPluginFile(text)
	if err != nil {
		return err
	}
	fmt.Printf("Metadata:\n%s\n", string(metadataBytes))
	fmt.Printf("Template:\n%s\n", string(templateBytes))

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
	textReader := bufio.NewReader(bytes.NewReader(text))
	var metadataBuf bytes.Buffer
	var templateBuf bytes.Buffer

	// Find the pipeline
	lines := []string{}
	for {
		line, err := textReader.ReadString('\n')
		if err != nil {
			return nil, nil, fmt.Errorf("plugin file is missing the pipeline block")
		}
		lines = append(lines, line)
		if matched, _ := regexp.MatchString(`^pipeline:`, line); matched {
			break
		}
	}

	// Include all empty and commented lines above pipeline in the pipeline half.
	// Skip the last line since we know it's the pipeline
	emptyRegexp := regexp.MustCompile(`^\s*$`)
	commentedRegexp := regexp.MustCompile(`^\s*#`)
	i := len(lines) - 1
	for ; i >= 0; i-- {
		line := lines[i]
		if !emptyRegexp.MatchString(line) && !commentedRegexp.MatchString(line) {
			break
		}
	}

	for _, line := range lines[:i] {
		metadataBuf.WriteString(line)
	}

	for _, line := range lines[i:] {
		templateBuf.WriteString(line)
	}

	if _, err := io.Copy(&templateBuf, textReader); err != nil {
		return nil, nil, err
	}

	return metadataBuf.Bytes(), templateBuf.Bytes(), nil
}

func NewPluginFromFile(path string) (*Plugin, error) {
	contents, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}

	id := strings.Split(filepath.Base(path), ".")[0]
	return NewPlugin(id, contents)
}

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
