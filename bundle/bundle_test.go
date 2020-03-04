package bundle

import (
	"html/template"
	"io/ioutil"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/xeipuuv/gojsonschema"
)

func TestBundleDefinitionRender(t *testing.T) {

	tmpl, err := template.New("config").Parse(`
plugins:
{{ if .enable }}
- id: enabled
  type: test
{{ end }}
{{ if .disable }}
- id: disabled
  type: test
{{ end }}`)
	assert.NoError(t, err)

	schemaLoader := gojsonschema.NewStringLoader(`{
  "type": "object",
  "properties": {
    "enable": {
      "type": "boolean"
    },
    "disable": {
      "type": "boolean"
    }
  }}`)
	schema, err := gojsonschema.NewSchema(schemaLoader)
	assert.NoError(t, err)

	def := BundleDefinition{
		spec:     schema,
		template: tmpl,
	}

	params := map[string]interface{}{
		"enable":  true,
		"disable": false,
	}

	configReader, err := def.Render(params)
	assert.NoError(t, err)

	config, err := ioutil.ReadAll(configReader)
	assert.NoError(t, err)

	expected := `
plugins:
- id: enabled
  type: test`
	assert.YAMLEq(t, expected, string(config))

}
