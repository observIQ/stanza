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
	"go.uber.org/zap"
)

type BundleDefinition struct {
	spec     *gojsonschema.Schema
	template *template.Template
}

// TODO find a more elegant way of logging than passing in a logger
func GetBundleDefinitions(bundleDir string, logger *zap.SugaredLogger) ([]*BundleDefinition, error) {
	bundleDefinitions := make([]*BundleDefinition, 0)
	err := filepath.Walk(bundleDir, newBundleWalkFunc(bundleDefinitions, logger))
	if err != nil {
		return nil, fmt.Errorf("failed to parse bundle definitions: %s", err)
	}

	return bundleDefinitions, nil
}

func newBundleWalkFunc(bundleDefinitions []*BundleDefinition, logger *zap.SugaredLogger) filepath.WalkFunc {
	return func(path string, info os.FileInfo, fileError error) error {
		if fileError != nil {
			logger.Warnw("File walker failed to process file", "error", fileError, "file", path)
			return nil
		}

		if info.IsDir() {
			return filepath.SkipDir
		}

		file, err := os.Open(path)
		if err != nil {
			logger.Warnw("Failed to open bundle file", "error", err, "file", path)
			return nil
		}
		defer file.Close()

		bundle, err := ParseBundle(file)
		if err != nil {
			logger.Warnw("Failed to parse file in bundle directory as bundle", "error", err, "file", path)
			return nil
		}

		bundleDefinitions = append(bundleDefinitions, bundle)
		return nil
	}
}

func ParseBundle(file io.Reader) (*BundleDefinition, error) {
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
	jsonLoader, reader := gojsonschema.NewReaderLoader(specReader)
	_, _ = ioutil.ReadAll(reader)

	schema, err := gojsonschema.NewSchema(jsonLoader)
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
