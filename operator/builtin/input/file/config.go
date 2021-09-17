package file

import (
	"fmt"
	"regexp"
	"time"

	"github.com/bmatcuk/doublestar/v2"
	"github.com/observiq/stanza/entry"
	"github.com/observiq/stanza/operator"
	"github.com/observiq/stanza/operator/helper"
)

func init() {
	operator.Register("file_input", func() operator.Builder { return NewInputConfig("") })
}

const (
	defaultMaxLogSize         = 1024 * 1024
	defaultMaxConcurrentFiles = 512
)

// NewInputConfig creates a new input config with default values
func NewInputConfig(operatorID string) *InputConfig {
	return &InputConfig{
		InputConfig:             helper.NewInputConfig(operatorID, "file_input"),
		PollInterval:            helper.Duration{Duration: 200 * time.Millisecond},
		IncludeFileName:         true,
		IncludeFilePath:         false,
		IncludeFileNameResolved: false,
		IncludeFilePathResolved: false,
		StartAt:                 "end",
		FingerprintSize:         defaultFingerprintSize,
		MaxLogSize:              defaultMaxLogSize,
		MaxConcurrentFiles:      defaultMaxConcurrentFiles,
		Encoding:                helper.NewEncodingConfig(),
	}
}

// InputConfig is the configuration of a file input operator
type InputConfig struct {
	helper.InputConfig `yaml:",inline"`

	Include []string `json:"include,omitempty" yaml:"include,omitempty"`
	Exclude []string `json:"exclude,omitempty" yaml:"exclude,omitempty"`

	PollInterval            helper.Duration        `json:"poll_interval,omitempty"               yaml:"poll_interval,omitempty"`
	Multiline               helper.MultilineConfig `json:"multiline,omitempty"                   yaml:"multiline,omitempty"`
	IncludeFileName         bool                   `json:"include_file_name,omitempty"           yaml:"include_file_name,omitempty"`
	IncludeFilePath         bool                   `json:"include_file_path,omitempty"           yaml:"include_file_path,omitempty"`
	IncludeFileNameResolved bool                   `json:"include_file_name_resolved,omitempty"  yaml:"include_file_name_resolved,omitempty"`
	IncludeFilePathResolved bool                   `json:"include_file_path_resolved,omitempty"  yaml:"include_file_path_resolved,omitempty"`
	StartAt                 string                 `json:"start_at,omitempty"                    yaml:"start_at,omitempty"`
	FingerprintSize         helper.ByteSize        `json:"fingerprint_size,omitempty"            yaml:"fingerprint_size,omitempty"`
	MaxLogSize              helper.ByteSize        `json:"max_log_size,omitempty"                yaml:"max_log_size,omitempty"`
	MaxConcurrentFiles      int                    `json:"max_concurrent_files,omitempty"        yaml:"max_concurrent_files,omitempty"`
	DeleteAfterRead         bool                   `json:"delete_after_read,omitempty"           yaml:"delete_after_read,omitempty"`
	AttributeRegex          string                 `json:"attribute_regex,omitempty"                 yaml:"attribute_regex,omitempty"`
	Encoding                helper.EncodingConfig  `json:",inline,omitempty"                     yaml:",inline,omitempty"`
}

// Build will build a file input operator from the supplied configuration
func (c InputConfig) Build(context operator.BuildContext) ([]operator.Operator, error) {
	inputOperator, err := c.InputConfig.Build(context)
	if err != nil {
		return nil, err
	}

	if len(c.Include) == 0 {
		return nil, fmt.Errorf("required argument `include` is empty")
	}

	// Ensure includes can be parsed as globs
	for _, include := range c.Include {
		_, err := doublestar.PathMatch(include, "matchstring")
		if err != nil {
			return nil, fmt.Errorf("parse include glob: %s", err)
		}
	}

	// Ensure excludes can be parsed as globs
	for _, exclude := range c.Exclude {
		_, err := doublestar.PathMatch(exclude, "matchstring")
		if err != nil {
			return nil, fmt.Errorf("parse exclude glob: %s", err)
		}
	}

	if c.MaxLogSize <= 0 {
		return nil, fmt.Errorf("`max_log_size` must be positive")
	}

	if c.MaxConcurrentFiles <= 1 {
		return nil, fmt.Errorf("`max_concurrent_files` must be greater than 1")
	}

	if c.FingerprintSize == 0 {
		c.FingerprintSize = defaultFingerprintSize
	} else if c.FingerprintSize < minFingerprintSize {
		return nil, fmt.Errorf("`fingerprint_size` must be at least %d bytes", minFingerprintSize)
	}

	encoding, err := c.Encoding.Build(context)
	if err != nil {
		return nil, err
	}

	splitFunc, err := c.Multiline.Build(context, encoding.Encoding, false)
	if err != nil {
		return nil, err
	}

	var startAtBeginning bool
	switch c.StartAt {
	case "beginning":
		startAtBeginning = true
	case "end":
		if c.DeleteAfterRead {
			return nil, fmt.Errorf("delete_after_read cannot be used with start_at 'end'")
		}
		startAtBeginning = false
	default:
		return nil, fmt.Errorf("invalid start_at location '%s'", c.StartAt)
	}

	var attributeRegex *regexp.Regexp
	if c.AttributeRegex != "" {
		r, err := regexp.Compile(c.AttributeRegex)
		if err != nil {
			return nil, fmt.Errorf("compiling regex: %s", err)
		}

		keys := r.SubexpNames()
		// keys[0] is always the empty string
		if x := len(keys); x != 3 {
			return nil, fmt.Errorf("attribute_regex must contain two capture groups named 'key' and 'value', got %d capture groups", x)
		}

		hasKeys := make(map[string]bool)
		hasKeys[keys[1]] = true
		hasKeys[keys[2]] = true
		if !hasKeys["key"] || !hasKeys["value"] {
			return nil, fmt.Errorf("attribute_regex must contain two capture groups named 'key' and 'value'")
		}
		attributeRegex = r
	}

	fileNameField := entry.NewNilField()
	if c.IncludeFileName {
		fileNameField = entry.NewAttributeField("file_name")
	}

	filePathField := entry.NewNilField()
	if c.IncludeFilePath {
		filePathField = entry.NewAttributeField("file_path")
	}

	fileNameResolvedField := entry.NewNilField()
	if c.IncludeFileNameResolved {
		fileNameResolvedField = entry.NewAttributeField("file_name_resolved")
	}

	filePathResolvedField := entry.NewNilField()
	if c.IncludeFilePathResolved {
		filePathResolvedField = entry.NewAttributeField("file_path_resolved")
	}

	op := &InputOperator{
		InputOperator:         inputOperator,
		Include:               c.Include,
		Exclude:               c.Exclude,
		SplitFunc:             splitFunc,
		PollInterval:          c.PollInterval.Raw(),
		persist:               helper.NewScopedDBPersister(context.Database, c.ID()),
		FilePathField:         filePathField,
		FileNameField:         fileNameField,
		FilePathResolvedField: filePathResolvedField,
		FileNameResolvedField: fileNameResolvedField,
		startAtBeginning:      startAtBeginning,
		deleteAfterRead:       c.DeleteAfterRead,
		queuedMatches:         make([]string, 0),
		attributeRegex:        attributeRegex,
		encoding:              encoding,
		firstCheck:            true,
		cancel:                func() {},
		knownFiles:            make([]*Reader, 0, 10),
		fingerprintSize:       int(c.FingerprintSize),
		MaxLogSize:            int(c.MaxLogSize),
		MaxConcurrentFiles:    c.MaxConcurrentFiles,
		SeenPaths:             make(map[string]struct{}, 100),
	}

	return []operator.Operator{op}, nil
}
