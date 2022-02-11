package service

import (
	"io/ioutil"
	"os"
	"testing"

	"github.com/open-telemetry/opentelemetry-log-collection/operator"
	"github.com/open-telemetry/opentelemetry-log-collection/operator/builtin/transformer/noop"
	"github.com/open-telemetry/opentelemetry-log-collection/operator/helper"
	"github.com/open-telemetry/opentelemetry-log-collection/pipeline"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zapcore"
)

func TestLoadConfig(t *testing.T) {
	testCases := []struct {
		name        string
		yaml        string
		expected    Config
		expectedErr string
	}{
		{
			name: "basic pipeline",
			yaml: `
pipeline:
- id: "my-operator"
  type: "noop"`,
			expected: Config{
				Pipeline: pipeline.Config{
					operator.Config{
						Builder: &noop.NoopOperatorConfig{
							TransformerConfig: helper.TransformerConfig{
								WriterConfig: helper.WriterConfig{
									BasicConfig: helper.BasicConfig{
										OperatorID:   "my-operator",
										OperatorType: "noop",
									},
								},
								OnError: "send",
							},
						},
					},
				},
				Logging: DefaultLoggingConfig(),
			},
		},
		{
			name: "minimal config",
			yaml: `pipeline:`,
			expected: Config{
				Logging: DefaultLoggingConfig(),
			},
		},
		{
			name: "stdout logging config",
			yaml: `
pipeline:
logging:
    output: stdout`,
			expected: Config{
				Logging: DefaultLoggingConfig(),
			},
		},
		{
			name: "file logging config",
			yaml: `
pipeline:
logging:
    output: file`,
			expected: Config{
				Logging: func() *LoggingConfig {
					lc := DefaultLoggingConfig()
					lc.Output = fileOutput
					return lc
				}(),
			},
		},
		{
			name: "file logging config options",
			yaml: `
pipeline:
logging:
    output: file
    file:
        filename: "example.log"
        maxbackups: 15
        maxsize: 16
        maxage: 17`,
			expected: Config{
				Logging: func() *LoggingConfig {
					lc := DefaultLoggingConfig()
					lc.Output = fileOutput
					lc.File.Filename = "example.log"
					lc.File.MaxBackups = 15
					lc.File.MaxSize = 16
					lc.File.MaxAge = 17
					return lc
				}(),
			},
		},
		{
			name: "info log level",
			yaml: `
pipeline:
logging:
    output: stdout
    level: info
`,
			expected: Config{
				Logging: func() *LoggingConfig {
					lc := DefaultLoggingConfig()
					lc.Level = zapcore.InfoLevel
					return lc
				}(),
			},
		},
		{
			name: "debug log level",
			yaml: `
pipeline:
logging:
    output: stdout
    level: debug
`,
			expected: Config{
				Logging: func() *LoggingConfig {
					lc := DefaultLoggingConfig()
					lc.Level = zapcore.DebugLevel
					return lc
				}(),
			},
		},
		{
			name: "error log level",
			yaml: `
pipeline:
logging:
    output: stdout
    level: error
`,
			expected: Config{
				Logging: func() *LoggingConfig {
					lc := DefaultLoggingConfig()
					lc.Level = zapcore.ErrorLevel
					return lc
				}(),
			},
		},
		{
			name: "unknown field in yaml",
			yaml: `
pipeline:
logging:
    some_field: a
`,
			expectedErr: "field some_field not found",
		},
	}

	for i := range testCases {
		testCase := testCases[i]
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			f, err := ioutil.TempFile("", "test-logging-config")
			require.NoError(t, err)
			path := f.Name()
			defer os.Remove(path)

			_, err = f.Write([]byte(testCase.yaml))
			require.NoError(t, err)

			err = f.Close()
			require.NoError(t, err)

			conf, err := LoadConfig(f.Name())

			if testCase.expectedErr == "" {
				require.NoError(t, err)
				require.Equal(t, testCase.expected, *conf)
			} else {
				require.Error(t, err)
				require.Contains(t, err.Error(), testCase.expectedErr)
			}
		})
	}
}
