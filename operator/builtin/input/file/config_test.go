// Copyright The OpenTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package file

import (
	"testing"
	"time"

	"github.com/observiq/stanza/entry"
	"github.com/observiq/stanza/operator"
	"github.com/observiq/stanza/operator/helper"
	"github.com/observiq/stanza/operator/helper/operatortest"
	"github.com/observiq/stanza/testutil"
	"github.com/stretchr/testify/require"
)

func TestUnmarshal(t *testing.T) {
	cases := []operatortest.ConfigUnmarshalTest{
		{
			Name:      "default",
			ExpectErr: false,
			Expect:    defaultCfg(),
		},
		{

			Name:      "extra_field",
			ExpectErr: false,
			Expect:    defaultCfg(),
		},
		{
			Name:      "id_custom",
			ExpectErr: false,
			Expect:    NewInputConfig("test_id"),
		},
		{
			Name:      "include_one",
			ExpectErr: false,
			Expect: func() *InputConfig {
				cfg := defaultCfg()
				cfg.Include = append(cfg.Include, "one.log")
				return cfg
			}(),
		},
		{
			Name:      "include_multi",
			ExpectErr: false,
			Expect: func() *InputConfig {
				cfg := defaultCfg()
				cfg.Include = append(cfg.Include, "one.log", "two.log", "three.log")
				return cfg
			}(),
		},
		{
			Name:      "include_glob",
			ExpectErr: false,
			Expect: func() *InputConfig {
				cfg := defaultCfg()
				cfg.Include = append(cfg.Include, "*.log")
				return cfg
			}(),
		},
		{
			Name:      "include_glob_double_asterisk",
			ExpectErr: false,
			Expect: func() *InputConfig {
				cfg := defaultCfg()
				cfg.Include = append(cfg.Include, "**.log")
				return cfg
			}(),
		},
		{
			Name:      "include_glob_double_asterisk_nested",
			ExpectErr: false,
			Expect: func() *InputConfig {
				cfg := defaultCfg()
				cfg.Include = append(cfg.Include, "directory/**/*.log")
				return cfg
			}(),
		},
		{
			Name:      "include_glob_double_asterisk_prefix",
			ExpectErr: false,
			Expect: func() *InputConfig {
				cfg := defaultCfg()
				cfg.Include = append(cfg.Include, "**/directory/**/*.log")
				return cfg
			}(),
		},
		{
			Name:      "include_inline",
			ExpectErr: false,
			Expect: func() *InputConfig {
				cfg := defaultCfg()
				cfg.Include = append(cfg.Include, "a.log", "b.log")
				return cfg
			}(),
		},
		{
			Name:      "include_invalid",
			ExpectErr: true,
			Expect:    nil,
		},
		{
			Name:      "exclude_one",
			ExpectErr: false,
			Expect: func() *InputConfig {
				cfg := defaultCfg()
				cfg.Include = append(cfg.Include, "*.log")
				cfg.Exclude = append(cfg.Exclude, "one.log")
				return cfg
			}(),
		},
		{
			Name:      "exclude_multi",
			ExpectErr: false,
			Expect: func() *InputConfig {
				cfg := defaultCfg()
				cfg.Include = append(cfg.Include, "*.log")
				cfg.Exclude = append(cfg.Exclude, "one.log", "two.log", "three.log")
				return cfg
			}(),
		},
		{
			Name:      "exclude_glob",
			ExpectErr: false,
			Expect: func() *InputConfig {
				cfg := defaultCfg()
				cfg.Include = append(cfg.Include, "*.log")
				cfg.Exclude = append(cfg.Exclude, "not*.log")
				return cfg
			}(),
		},
		{
			Name:      "exclude_glob_double_asterisk",
			ExpectErr: false,
			Expect: func() *InputConfig {
				cfg := defaultCfg()
				cfg.Include = append(cfg.Include, "*.log")
				cfg.Exclude = append(cfg.Exclude, "not**.log")
				return cfg
			}(),
		},
		{
			Name:      "exclude_glob_double_asterisk_nested",
			ExpectErr: false,
			Expect: func() *InputConfig {
				cfg := defaultCfg()
				cfg.Include = append(cfg.Include, "*.log")
				cfg.Exclude = append(cfg.Exclude, "directory/**/not*.log")
				return cfg
			}(),
		},
		{
			Name:      "exclude_glob_double_asterisk_prefix",
			ExpectErr: false,
			Expect: func() *InputConfig {
				cfg := defaultCfg()
				cfg.Include = append(cfg.Include, "*.log")
				cfg.Exclude = append(cfg.Exclude, "**/directory/**/not*.log")
				return cfg
			}(),
		},
		{
			Name:      "exclude_inline",
			ExpectErr: false,
			Expect: func() *InputConfig {
				cfg := defaultCfg()
				cfg.Include = append(cfg.Include, "*.log")
				cfg.Exclude = append(cfg.Exclude, "a.log", "b.log")
				return cfg
			}(),
		},
		{
			Name:      "exclude_invalid",
			ExpectErr: true,
			Expect:    nil,
		},
		{
			Name:      "poll_interval_no_units",
			ExpectErr: false,
			Expect: func() *InputConfig {
				cfg := defaultCfg()
				cfg.PollInterval = helper.NewDuration(time.Second)
				return cfg
			}(),
		},
		{
			Name:      "poll_interval_1s",
			ExpectErr: false,
			Expect: func() *InputConfig {
				cfg := defaultCfg()
				cfg.PollInterval = helper.NewDuration(time.Second)
				return cfg
			}(),
		},
		{
			Name:      "poll_interval_1ms",
			ExpectErr: false,
			Expect: func() *InputConfig {
				cfg := defaultCfg()
				cfg.PollInterval = helper.NewDuration(time.Millisecond)
				return cfg
			}(),
		},
		{
			Name:      "poll_interval_1000ms",
			ExpectErr: false,
			Expect: func() *InputConfig {
				cfg := defaultCfg()
				cfg.PollInterval = helper.NewDuration(time.Second)
				return cfg
			}(),
		},
		{
			Name:      "fingerprint_size_no_units",
			ExpectErr: false,
			Expect: func() *InputConfig {
				cfg := defaultCfg()
				cfg.FingerprintSize = helper.ByteSize(1000)
				return cfg
			}(),
		},
		{
			Name:      "fingerprint_size_1kb_lower",
			ExpectErr: false,
			Expect: func() *InputConfig {
				cfg := defaultCfg()
				cfg.FingerprintSize = helper.ByteSize(1000)
				return cfg
			}(),
		},
		{
			Name:      "fingerprint_size_1KB",
			ExpectErr: false,
			Expect: func() *InputConfig {
				cfg := defaultCfg()
				cfg.FingerprintSize = helper.ByteSize(1000)
				return cfg
			}(),
		},
		{
			Name:      "fingerprint_size_1kib_lower",
			ExpectErr: false,
			Expect: func() *InputConfig {
				cfg := defaultCfg()
				cfg.FingerprintSize = helper.ByteSize(1024)
				return cfg
			}(),
		},
		{
			Name:      "fingerprint_size_1KiB",
			ExpectErr: false,
			Expect: func() *InputConfig {
				cfg := defaultCfg()
				cfg.FingerprintSize = helper.ByteSize(1024)
				return cfg
			}(),
		},
		{
			Name:      "fingerprint_size_float",
			ExpectErr: false,
			Expect: func() *InputConfig {
				cfg := defaultCfg()
				cfg.FingerprintSize = helper.ByteSize(1100)
				return cfg
			}(),
		},
		{
			Name:      "include_file_name_lower",
			ExpectErr: false,
			Expect: func() *InputConfig {
				cfg := defaultCfg()
				cfg.Include = append(cfg.Include, "one.log")
				cfg.IncludeFileName = true
				return cfg
			}(),
		},
		{
			Name:      "include_file_name_upper",
			ExpectErr: false,
			Expect: func() *InputConfig {
				cfg := defaultCfg()
				cfg.Include = append(cfg.Include, "one.log")
				cfg.IncludeFileName = true
				return cfg
			}(),
		},
		{
			Name:      "include_file_name_on",
			ExpectErr: false,
			Expect: func() *InputConfig {
				cfg := defaultCfg()
				cfg.Include = append(cfg.Include, "one.log")
				cfg.IncludeFileName = true
				return cfg
			}(),
		},
		{
			Name:      "include_file_name_yes",
			ExpectErr: false,
			Expect: func() *InputConfig {
				cfg := defaultCfg()
				cfg.Include = append(cfg.Include, "one.log")
				cfg.IncludeFileName = true
				return cfg
			}(),
		},
		{
			Name:      "include_file_path_lower",
			ExpectErr: false,
			Expect: func() *InputConfig {
				cfg := defaultCfg()
				cfg.Include = append(cfg.Include, "one.log")
				cfg.IncludeFilePath = true
				return cfg
			}(),
		},
		{
			Name:      "include_file_path_upper",
			ExpectErr: false,
			Expect: func() *InputConfig {
				cfg := defaultCfg()
				cfg.Include = append(cfg.Include, "one.log")
				cfg.IncludeFilePath = true
				return cfg
			}(),
		},
		{
			Name:      "include_file_path_on",
			ExpectErr: false,
			Expect: func() *InputConfig {
				cfg := defaultCfg()
				cfg.Include = append(cfg.Include, "one.log")
				cfg.IncludeFilePath = true
				return cfg
			}(),
		},
		{
			Name:      "include_file_path_yes",
			ExpectErr: false,
			Expect: func() *InputConfig {
				cfg := defaultCfg()
				cfg.Include = append(cfg.Include, "one.log")
				cfg.IncludeFilePath = true
				return cfg
			}(),
		},
		{
			Name:      "include_file_path_off",
			ExpectErr: false,
			Expect: func() *InputConfig {
				cfg := defaultCfg()
				cfg.Include = append(cfg.Include, "one.log")
				cfg.IncludeFilePath = false
				return cfg
			}(),
		},
		{
			Name:      "include_file_path_no",
			ExpectErr: false,
			Expect: func() *InputConfig {
				cfg := defaultCfg()
				cfg.Include = append(cfg.Include, "one.log")
				cfg.IncludeFilePath = false
				return cfg
			}(),
		},
		{
			Name:      "include_file_path_nonbool",
			ExpectErr: true,
			Expect:    nil,
		},
		{
			Name:      "multiline_line_start_string",
			ExpectErr: false,
			Expect: func() *InputConfig {
				cfg := defaultCfg()
				newMulti := helper.MultilineConfig{}
				newMulti.LineStartPattern = "Start"
				cfg.Multiline = newMulti
				return cfg
			}(),
		},
		{
			Name:      "multiline_line_start_special",
			ExpectErr: false,
			Expect: func() *InputConfig {
				cfg := defaultCfg()
				newMulti := helper.MultilineConfig{}
				newMulti.LineStartPattern = "%"
				cfg.Multiline = newMulti
				return cfg
			}(),
		},
		{
			Name:      "multiline_line_end_string",
			ExpectErr: false,
			Expect: func() *InputConfig {
				cfg := defaultCfg()
				newMulti := helper.MultilineConfig{}
				newMulti.LineEndPattern = "Start"
				cfg.Multiline = newMulti
				return cfg
			}(),
		},
		{
			Name:      "multiline_line_end_special",
			ExpectErr: false,
			Expect: func() *InputConfig {
				cfg := defaultCfg()
				newMulti := helper.MultilineConfig{}
				newMulti.LineEndPattern = "%"
				cfg.Multiline = newMulti
				return cfg
			}(),
		},
		{
			Name:      "multiline_random",
			ExpectErr: true,
			Expect:    nil,
		},
		{
			Name:      "start_at_string",
			ExpectErr: false,
			Expect: func() *InputConfig {
				cfg := defaultCfg()
				cfg.StartAt = "beginning"
				return cfg
			}(),
		},
		{
			Name:      "max_concurrent_large",
			ExpectErr: false,
			Expect: func() *InputConfig {
				cfg := defaultCfg()
				cfg.MaxConcurrentFiles = 9223372036854775807
				return cfg
			}(),
		},
		{
			Name:      "max_log_size_mib_lower",
			ExpectErr: false,
			Expect: func() *InputConfig {
				cfg := defaultCfg()
				cfg.MaxLogSize = helper.ByteSize(1048576)
				return cfg
			}(),
		},
		{
			Name:      "max_log_size_mib_upper",
			ExpectErr: false,
			Expect: func() *InputConfig {
				cfg := defaultCfg()
				cfg.MaxLogSize = helper.ByteSize(1048576)
				return cfg
			}(),
		},
		{
			Name:      "max_log_size_mb_upper",
			ExpectErr: false,
			Expect: func() *InputConfig {
				cfg := defaultCfg()
				cfg.MaxLogSize = helper.ByteSize(1048576)
				return cfg
			}(),
		},
		{
			Name:      "max_log_size_mb_lower",
			ExpectErr: false,
			Expect: func() *InputConfig {
				cfg := defaultCfg()
				cfg.MaxLogSize = helper.ByteSize(1048576)
				return cfg
			}(),
		},
		{
			Name:      "encoding_lower",
			ExpectErr: false,
			Expect: func() *InputConfig {
				cfg := defaultCfg()
				cfg.Encoding = helper.EncodingConfig{Encoding: "utf-16le"}
				return cfg
			}(),
		},
		{
			Name:      "encoding_upper",
			ExpectErr: false,
			Expect: func() *InputConfig {
				cfg := defaultCfg()
				cfg.Encoding = helper.EncodingConfig{Encoding: "UTF-16lE"}
				return cfg
			}(),
		},
		{
			Name:      "label_regex",
			ExpectErr: false,
			Expect: func() *InputConfig {
				cfg := defaultCfg()
				cfg.LabelRegex = "^(?P<key>[a-zA-z]+ [A-Z]+): (?P<value>.*)"
				return cfg
			}(),
		},
	}

	for _, tc := range cases {
		t.Run(tc.Name, func(t *testing.T) {
			tc.Run(t, defaultCfg())
		})
	}
}

func TestBuild(t *testing.T) {
	t.Parallel()
	fakeOutput := testutil.NewMockOperator("$.fake")

	basicConfig := func() *InputConfig {
		cfg := NewInputConfig("testfile")
		cfg.OutputIDs = []string{"fake"}
		cfg.Include = []string{"/var/log/testpath.*"}
		cfg.Exclude = []string{"/var/log/testpath.ex*"}
		cfg.PollInterval = helper.Duration{Duration: 10 * time.Millisecond}
		return cfg
	}

	cases := []struct {
		name             string
		modifyBaseConfig func(*InputConfig)
		errorRequirement require.ErrorAssertionFunc
		validate         func(*testing.T, *InputOperator)
	}{
		{
			"Basic",
			func(f *InputConfig) { return },
			require.NoError,
			func(t *testing.T, f *InputOperator) {
				require.Equal(t, f.OutputOperators[0], fakeOutput)
				require.Equal(t, f.Include, []string{"/var/log/testpath.*"})
				require.Equal(t, f.FilePathField, entry.NewNilField())
				require.Equal(t, f.FileNameField, entry.NewLabelField("file_name"))
				require.Equal(t, f.PollInterval, 10*time.Millisecond)
				require.Equal(t, f.MaxConcurrentFiles, defaultMaxConcurrentFiles)
			},
		},
		{
			"MaxConcurrentFilesCustom",
			func(f *InputConfig) {
				f.MaxConcurrentFiles = 100
				return
			},
			require.NoError,
			func(t *testing.T, f *InputOperator) {
				require.Equal(t, f.OutputOperators[0], fakeOutput)
				require.Equal(t, f.Include, []string{"/var/log/testpath.*"})
				require.Equal(t, f.FilePathField, entry.NewNilField())
				require.Equal(t, f.FileNameField, entry.NewLabelField("file_name"))
				require.Equal(t, f.PollInterval, 10*time.Millisecond)
				require.Equal(t, f.MaxConcurrentFiles, 100)
			},
		},
		{
			"MaxConcurrentFilesInvalid",
			func(f *InputConfig) {
				f.MaxConcurrentFiles = 1 // must be at least 2
			},
			require.Error,
			nil,
		},
		{
			"BadIncludeGlob",
			func(f *InputConfig) {
				f.Include = []string{"["}
			},
			require.Error,
			nil,
		},
		{
			"BadExcludeGlob",
			func(f *InputConfig) {
				f.Include = []string{"["}
			},
			require.Error,
			nil,
		},
		{
			"MultilineConfiguredStartAndEndPatterns",
			func(f *InputConfig) {
				f.Multiline = helper.MultilineConfig{
					LineEndPattern:   "Exists",
					LineStartPattern: "Exists",
				}
			},
			require.Error,
			nil,
		},
		{
			"MultilineConfiguredStartPattern",
			func(f *InputConfig) {
				f.Multiline = helper.MultilineConfig{
					LineStartPattern: "START.*",
				}
			},
			require.NoError,
			func(t *testing.T, f *InputOperator) {},
		},
		{
			"MultilineConfiguredEndPattern",
			func(f *InputConfig) {
				f.Multiline = helper.MultilineConfig{
					LineEndPattern: "END.*",
				}
			},
			require.NoError,
			func(t *testing.T, f *InputOperator) {},
		},
		{
			"InvalidEncoding",
			func(f *InputConfig) {
				f.Encoding = helper.EncodingConfig{Encoding: "UTF-3233"}
			},
			require.Error,
			nil,
		},
		{
			"LineStartAndEnd",
			func(f *InputConfig) {
				f.Multiline = helper.MultilineConfig{
					LineStartPattern: ".*",
					LineEndPattern:   ".*",
				}
			},
			require.Error,
			nil,
		},
		{
			"NoLineStartOrEnd",
			func(f *InputConfig) {
				f.Multiline = helper.MultilineConfig{}
			},
			require.NoError,
			func(t *testing.T, f *InputOperator) {},
		},
		{
			"InvalidLineStartRegex",
			func(f *InputConfig) {
				f.Multiline = helper.MultilineConfig{
					LineStartPattern: "(",
				}
			},
			require.Error,
			nil,
		},
		{
			"InvalidLineEndRegex",
			func(f *InputConfig) {
				f.Multiline = helper.MultilineConfig{
					LineEndPattern: "(",
				}
			},
			require.Error,
			nil,
		},
		{
			"ValidLabelRegex",
			func(f *InputConfig) {
				f.LabelRegex = "^(?P<key>[a-zA-z]+ [A-Z]+): (?P<value>.*)"
			},
			require.NoError,
			func(t *testing.T, f *InputOperator) {},
		},
		{
			"ValidLabelRegexReverse",
			func(f *InputConfig) {
				f.LabelRegex = "^(?P<value>[a-zA-z]+ [A-Z]+): (?P<key>.*)"
			},
			require.NoError,
			func(t *testing.T, f *InputOperator) {},
		},
		{
			"InvalidLabelRegexPattern",
			func(f *InputConfig) {
				f.LabelRegex = "^(?P<something>[a-zA-z]"
			},
			require.Error,
			nil,
		},
		{
			"InvalidLabelRegexCaptureGroup",
			func(f *InputConfig) {
				f.LabelRegex = "^(?P<something>[a-zA-z]+ [A-Z]+): (?P<invalid>.*)"
			},
			require.Error,
			nil,
		},
		{
			"InvalidStartAtDelete",
			func(f *InputConfig) {
				f.StartAt = "end"
				f.DeleteAfterRead = true
			},
			require.Error,
			nil,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			tc := tc
			t.Parallel()
			cfg := basicConfig()
			tc.modifyBaseConfig(cfg)

			ops, err := cfg.Build(testutil.NewBuildContext(t))
			tc.errorRequirement(t, err)
			if err != nil {
				return
			}
			op := ops[0]

			err = op.SetOutputs([]operator.Operator{fakeOutput})
			require.NoError(t, err)

			fileInput := op.(*InputOperator)
			tc.validate(t, fileInput)
		})
	}
}
func defaultCfg() *InputConfig {
	return NewInputConfig("file_input")
}

func NewTestInputConfig() *InputConfig {
	cfg := NewInputConfig("config_test")
	cfg.WriteTo = entry.Field{}
	cfg.Include = []string{"i1", "i2"}
	cfg.Exclude = []string{"e1", "e2"}
	cfg.Multiline = helper.MultilineConfig{
		LineStartPattern: "start",
		LineEndPattern:   "end",
	}
	cfg.FingerprintSize = 1024
	cfg.Encoding = helper.EncodingConfig{Encoding: "utf16"}
	return cfg
}
