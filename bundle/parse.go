package bundle

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/xeipuuv/gojsonschema"
)

func parseUncompressedBundle(dir string) (*BundleDefinition, error) {
	var schema *gojsonschema.Schema
	var tmpl *template.Template

	var def *BundleDefinition
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
				return err
			}
			defer file.Close()
			def = &BundleDefinition{}
			err = parseBundleSpec(file, def)
			if err != nil {
				return fmt.Errorf("parse spec.json as a bundle spec: %s", err)
			}
		case "schema.json":
			file, err := os.Open(path)
			if err != nil {
				return err
			}
			defer file.Close()
			schema, err = parseBundleSchema(file)
			if err != nil {
				return fmt.Errorf("parse schema.json as a bundle spec: %s", err)
			}
		case "config.tmpl":
			file, err := os.Open(path)
			if err != nil {
				return err
			}
			defer file.Close()
			tmpl, err = parseBundleTemplate(file)
			if err != nil {
				return fmt.Errorf("parse config.tmpl as a go template: %s", err)
			}
		default:
			return fmt.Errorf("bundle contains an unknown file '%s'", info.Name())
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	if schema == nil {
		return nil, fmt.Errorf("no schema.json found in bundle")
	}

	if tmpl == nil {
		return nil, fmt.Errorf("no config.tmpl found in bundle")
	}

	if def == nil {
		return nil, fmt.Errorf("no spec.json found in bundle")
	}

	def.schema = schema
	def.template = tmpl

	return def, nil
}

// TODO there is a bunch of repeated code between this and parseUncompressedBundle. Fix that
func parseCompressedBundle(file io.Reader) (*BundleDefinition, error) {
	decompressed, err := gzip.NewReader(file)
	if err != nil {
		return nil, fmt.Errorf("decode file as gzip: %s", err)
	}

	tarReader := tar.NewReader(decompressed)

	var schema *gojsonschema.Schema
	var tmpl *template.Template

	var def *BundleDefinition
	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("parse file as tar: %s", err)
		}

		switch header.Name {
		case "spec.json":
			def = &BundleDefinition{}
			err = parseBundleSpec(tarReader, def)
			if err != nil {
				return nil, fmt.Errorf("parse spec.json as a bundle spec: %s", err)
			}
		case "schema.json":
			schema, err = parseBundleSchema(tarReader)
			if err != nil {
				return nil, fmt.Errorf("parse schema.json as a bundle spec: %s", err)
			}
		case "config.tmpl":
			tmpl, err = parseBundleTemplate(tarReader)
			if err != nil {
				return nil, fmt.Errorf("parse config.tmpl as a go template: %s", err)
			}
		default:
			return nil, fmt.Errorf("bundle contains an unknown file '%s'", header.Name)
		}
	}

	if schema == nil {
		return nil, fmt.Errorf("no schema.json found in bundle")
	}

	if tmpl == nil {
		return nil, fmt.Errorf("no config.tmpl found in bundle")
	}

	if def == nil {
		return nil, fmt.Errorf("no spec.json found in bundle")
	}

	def.schema = schema
	def.template = tmpl

	return def, nil
}

func parseBundleSchema(specReader io.Reader) (*gojsonschema.Schema, error) {
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
		return nil, fmt.Errorf("read template file to string: %s", err)
	}
	templateString := buf.String()
	t := template.New("config")
	return t.Parse(templateString)
}

func parseBundleSpec(specReader io.Reader, dest *BundleDefinition) error {
	specJson, err := ioutil.ReadAll(specReader)
	if err != nil {
		return fmt.Errorf("read full spec: %s", err)
	}

	err = json.Unmarshal(specJson, dest)
	if err != nil {
		return fmt.Errorf("unmarshal spec: %s", err)
	}

	return nil
}
