package bundle

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"testing"

	"github.com/stretchr/testify/assert"
)

const simpleSchema = `{
  "title": "Example Schema",
  "type": "object",
  "properties": {
    "firstName": {
      "type": "string"
    },
    "lastName": {
      "type": "string"
    },
    "age": {
      "description": "Age in years",
      "type": "integer",
      "minimum": 0
    }
  },
  "required": ["firstName", "lastName"]
}`

func TestParseBundle(t *testing.T) {
	tmpl := `{{.Test}}`
	var buf bytes.Buffer
	gzWriter := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gzWriter)
	var files = []struct {
		Name, Body string
	}{
		{"spec.json", simpleSchema},
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

	_, err = ParseBundle(&buf)
	assert.NoError(t, err)

}
