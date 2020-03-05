package bundle

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"html/template"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/xeipuuv/gojsonschema"
)

const simpleSchema = `{
  "title": "Simple Schema",
  "type": "object",
  "properties": {
    "enabled": {
      "type": "boolean"
    },
    "value": {
      "type": "string"
    }
  },
  "required": ["enabled", "value"]
}`

const simpleSpec = `{
  "bundle_type": "simple",
  "is_inputter": false,
  "is_outputter": true
}`

const simpleTemplate = `
plugins:
{{if .enabled}}
- id: mygenerator
  type: generate
  count: 3
  record:
    testkey: {{.value}}
{{end}}
`

func newFakeBundleDefinition() *BundleDefinition {
	tmpl, _ := template.New("config").Parse(simpleTemplate)
	schemaLoader := gojsonschema.NewStringLoader(simpleSchema)
	schema, _ := gojsonschema.NewSchema(schemaLoader)
	return &BundleDefinition{
		BundleType:  "simple",
		InputID:     "",
		IsOutputter: true,

		schema:   schema,
		template: tmpl,
	}
}

func TestParseCompressedBundle_RoundTrip(t *testing.T) {
	tmpl := `{{.Test}}`
	var buf bytes.Buffer
	gzWriter := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gzWriter)
	var files = []struct {
		Name, Body string
	}{
		{"schema.json", simpleSchema},
		{"spec.json", simpleSpec},
		{"config.tmpl", tmpl},
	}
	for _, file := range files {
		hdr := &tar.Header{
			Name: file.Name,
			Mode: 0600,
			Size: int64(len(file.Body)),
		}
		if err := tw.WriteHeader(hdr); err != nil {
			assert.NoError(t, err)
		}
		if _, err := tw.Write([]byte(file.Body)); err != nil {
			assert.NoError(t, err)
		}
	}
	if err := tw.Close(); err != nil {
		assert.NoError(t, err)
	}
	err := gzWriter.Close()
	assert.NoError(t, err)

	_, err = parseCompressedBundle(&buf)
	assert.NoError(t, err)

}

func TestParseUncompressedBundle(t *testing.T) {
	// TODO test contents too
	_, err := parseUncompressedBundle("./test/sample_bundle")
	assert.NoError(t, err)
}

func TestParseCompressedBundle(t *testing.T) {
	// TODO test contents too
	file, err := os.Open("./test/sample_bundle.tar.gz")
	assert.NoError(t, err)
	defer file.Close()

	_, err = parseCompressedBundle(file)
	assert.NoError(t, err)
}
