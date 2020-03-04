package bundle

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"fmt"
	"html/template"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/xeipuuv/gojsonschema"
)

func parseUncompressedBundle(dir string) (*BundleDefinition, error) {
	var spec *gojsonschema.Schema
	var tmpl *template.Template

	err := filepath.Walk(dir, func(path string, info os.FileInfo, fileError error) error {
		if path == dir {
			return nil
		}

		if info.IsDir() {
			return fmt.Errorf("unexpected directory in bundle")
		}

		switch info.Name() {
		case "spec.json":
			file, err := os.Open(path)
			if err != nil {
				return fmt.Errorf("failed to open spec.json: %s", err)
			}
			defer file.Close()
			spec, err = parseBundleSpec(file)
			if err != nil {
				return fmt.Errorf("failed to parse spec.json as a bundle spec: %s", err)
			}
		case "config.tmpl":
			file, err := os.Open(path)
			if err != nil {
				return fmt.Errorf("failed to open spec.json: %s", err)
			}
			defer file.Close()
			tmpl, err = parseBundleTemplate(file)
			if err != nil {
				return fmt.Errorf("failed to parse config.tmpl as a go template: %s", err)
			}
		default:
			return fmt.Errorf("bundle contains an unknown file '%s'", info.Name())
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	if spec == nil {
		return nil, fmt.Errorf("no spec.json found in bundle")
	}

	if tmpl == nil {
		return nil, fmt.Errorf("no config.tmpl found in bundle")
	}

	def := BundleDefinition{
		spec:     spec,
		template: tmpl,
	}

	return &def, nil
}

func parseCompressedBundle(file io.Reader) (*BundleDefinition, error) {
	decompressed, err := gzip.NewReader(file)
	if err != nil {
		return nil, fmt.Errorf("failed to decode file as gzip: %s", err)
	}

	tarReader := tar.NewReader(decompressed)

	var spec *gojsonschema.Schema
	var tmpl *template.Template

	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("failed to parse file as tar: %s", err)
		}

		switch header.Name {
		case "spec.json":
			spec, err = parseBundleSpec(tarReader)
			if err != nil {
				return nil, fmt.Errorf("failed to parse spec.json as a bundle spec: %s", err)
			}

		case "config.tmpl":
			tmpl, err = parseBundleTemplate(tarReader)
			if err != nil {
				return nil, fmt.Errorf("failed to parse config.tmpl as a go template: %s", err)
			}
		default:
			return nil, fmt.Errorf("bundle contains an unknown file '%s'", header.Name)
		}
	}

	if spec == nil {
		return nil, fmt.Errorf("no spec.json found in bundle")
	}

	if tmpl == nil {
		return nil, fmt.Errorf("no config.tmpl found in bundle")
	}

	def := BundleDefinition{
		spec:     spec,
		template: tmpl,
	}

	return &def, nil
}

func parseBundleSpec(specReader io.Reader) (*gojsonschema.Schema, error) {
	specBytes, err := ioutil.ReadAll(specReader)
	if err != nil {
		return &gojsonschema.Schema{}, err
	}

	loader := gojsonschema.NewBytesLoader(specBytes)
	schema, err := gojsonschema.NewSchema(loader)
	if err != nil {
		return &gojsonschema.Schema{}, err
	}

	return schema, nil
}

func parseBundleTemplate(templateReader io.Reader) (*template.Template, error) {
	buf := new(bytes.Buffer)
	_, err := buf.ReadFrom(templateReader)
	if err != nil {
		return nil, fmt.Errorf("failed reading template file to string: %s", err)
	}
	templateString := buf.String()
	t := template.New("config")
	return t.Parse(templateString)
}
