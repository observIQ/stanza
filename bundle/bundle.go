package bundle

import (
	"bytes"
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
	// TODO should this be yaml rather than JSON?
	BundleType string `json:"bundle_type"`

	schema   *gojsonschema.Schema
	template *template.Template
}

func (def *BundleDefinition) Render(params map[string]interface{}) (io.Reader, error) {
	err := def.Validate(params)
	if err != nil {
		return nil, fmt.Errorf("validate params: %s", err)
	}

	var buf bytes.Buffer
	err = def.template.Execute(&buf, params)
	if err != nil {
		return nil, fmt.Errorf("render template: %s", err)
	}

	return &buf, nil
}

func (def *BundleDefinition) Validate(params map[string]interface{}) error {
	paramsLoader := gojsonschema.NewGoLoader(params)
	result, err := def.schema.Validate(paramsLoader)
	if err != nil {
		return fmt.Errorf("run schema validation: %s", err)
	}

	if !result.Valid() {
		return fmt.Errorf("validation failed with errors: %v", result.Errors())
	}

	return nil
}

func GetBundleDefinitions(bundleDir string, logger *zap.SugaredLogger) []*BundleDefinition {
	bundleDefinitions := make([]*BundleDefinition, 0)
	files, err := ioutil.ReadDir(bundleDir)
	if err != nil {
		logger.Errorw("Failed to get list of files", "error", err, "bundle_path", bundleDir)
	}

	for _, file := range files {
		path := filepath.Join(bundleDir, file.Name())

		// If it's a directory, try to parse it uncompressed
		if file.IsDir() {
			bundle, err := parseUncompressedBundle(path)
			if err != nil {
				logger.Warnw("Failed to parse directory in bundle directory as bundle", "error", err, "file", file.Name())
				continue
			}
			bundleDefinitions = append(bundleDefinitions, bundle)
			continue
		}

		// Otherwise, try to parse it as a .tar.gz
		file, err := os.Open(path)
		if err != nil {
			logger.Warnw("Failed to open bundle file", "error", err, "file", path)
			continue
		}
		defer file.Close()

		bundle, err := parseCompressedBundle(file)
		if err != nil {
			logger.Warnw("Failed to parse file in bundle directory as bundle", "error", err, "file", path)
			continue
		}

		bundleDefinitions = append(bundleDefinitions, bundle)
	}

	return bundleDefinitions
}
