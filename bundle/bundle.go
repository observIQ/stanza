package bundle

import (
	"bytes"
	"fmt"
	"html/template"
	"io"
	"os"
	"path/filepath"

	"github.com/xeipuuv/gojsonschema"
	"go.uber.org/zap"
)

type BundleDefinition struct {
	spec     *gojsonschema.Schema
	template *template.Template
}

func (def *BundleDefinition) Render(params map[string]interface{}) (io.Reader, error) {
	err := def.Validate(params)
	if err != nil {
		return nil, fmt.Errorf("failed to validate params: %s", err)
	}

	var buf bytes.Buffer
	err = def.template.Execute(&buf, params)
	if err != nil {
		return nil, fmt.Errorf("failed to render template: %s", err)
	}

	return &buf, nil
}

func (def *BundleDefinition) Validate(params map[string]interface{}) error {
	paramsLoader := gojsonschema.NewGoLoader(params)
	result, err := def.spec.Validate(paramsLoader)
	if err != nil {
		return fmt.Errorf("failed to run schema validation: %s", err)
	}

	if !result.Valid() {
		return fmt.Errorf("validation failed with errors: %v", result.Errors())
	}

	return nil
}

// TODO find a more elegant way of logging than passing in a logger
func GetBundleDefinitions(bundleDir string, logger *zap.SugaredLogger) []*BundleDefinition {
	bundleDefinitions := make([]*BundleDefinition, 0)
	err := filepath.Walk(bundleDir, newBundleWalkFunc(bundleDefinitions, logger))
	if err != nil {
		// TODO do we actually want to be able to throw an error, or should we just log?
		panic(err)
	}

	return bundleDefinitions
}

func newBundleWalkFunc(bundleDefinitions []*BundleDefinition, logger *zap.SugaredLogger) filepath.WalkFunc {
	return func(path string, info os.FileInfo, fileError error) error {
		if fileError != nil {
			logger.Warnw("File walker failed to process file", "error", fileError, "file", path)
			return nil
		}

		if info.IsDir() {
			bundle, err := parseUncompressedBundle(path)
			if err != nil {
				logger.Warnw("Failed to parse directory in bundle directory as bundle", "error", err, "file", path)
				return nil
			}
			bundleDefinitions = append(bundleDefinitions, bundle)
			return nil
		}

		file, err := os.Open(path)
		if err != nil {
			logger.Warnw("Failed to open bundle file", "error", err, "file", path)
			return nil
		}
		defer file.Close()

		bundle, err := parseCompressedBundle(file)
		if err != nil {
			logger.Warnw("Failed to parse file in bundle directory as bundle", "error", err, "file", path)
			return nil
		}

		bundleDefinitions = append(bundleDefinitions, bundle)
		return nil
	}
}
