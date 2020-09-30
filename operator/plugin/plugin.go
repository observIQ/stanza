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
func (p *Plugin) NewBuilder() operator.Builder {
	return &Config{
		plugin: p,
	}
}

// Render will render a plugin's template with the given parameters
func (p *Plugin) Render(pluginType string, params map[string]interface{}) ([]byte, error) {
	if err := p.Validate(params); err != nil {
		return nil, err
	}

	var writer bytes.Buffer
	if err := p.Template.Execute(&writer, params); err != nil {
		return nil, errors.NewError(
			"failed to render template for plugin",
			"ensure that all parameters are valid for the plugin",
			"plugin_type", pluginType,
			"error_message", err.Error(),
		)
	}

	return writer.Bytes(), nil
}

func (p *Plugin) Validate(params map[string]interface{}) error {
	for name, param := range p.Parameters {
		if err := param.validateDefintion(); err != nil {
			return errors.NewError(
				"invalid parameter found in plugin",
				"ensure that all parameters are valid for the plugin",
				"plugin_type", p.ID,
				"plugin_parameter", name,
				"error_message", err.Error(),
			)
		}

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

	if err := yaml.Unmarshal(metadataBytes, p); err != nil {
		return err
	}

	p.Template, err = template.New(p.Title).Parse(string(templateBytes))
	return err
}

func splitPluginFile(text []byte) (metadata, template []byte, err error) {
	// Split the file into the metadata and the template by looknig for the first
	// unindented line after `parameters`
	textReader := bufio.NewReader(bytes.NewReader(text))
	var metadataBuf bytes.Buffer
	var templateBuf bytes.Buffer

	// Find parameters line
	for {
		line, err := textReader.ReadString('\n')
		if err != nil {
			return nil, nil, fmt.Errorf("plugin file is missing the parameters block")
		}
		if _, err := metadataBuf.WriteString(line); err != nil {
			return nil, nil, err
		}
		if matched, _ := regexp.MatchString(`^parameters:`, line); matched {
			break
		}
	}

	// Find the next unindented line
	for {
		line, err := textReader.ReadString('\n')
		if err != nil {
			return nil, nil, fmt.Errorf("plugin file is missing a template after the parameters block")
		}
		if indented, _ := regexp.MatchString(`^\s+`, line); !indented {
			templateBuf.WriteString(line)
			break
		}
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

	pluginID := strings.Split(path, ".")[0]
	p := &Plugin{
		ID: pluginID,
	}
	if err = p.UnmarshalText(contents); err != nil {
		return nil, err
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
