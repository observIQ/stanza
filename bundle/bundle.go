package bundle

import (
	"fmt"
	"html/template"
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
