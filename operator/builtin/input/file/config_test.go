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
	"github.com/observiq/stanza/operator/helper"
	"github.com/observiq/stanza/operator/helper/operatortest"
)

func TestConfig(t *testing.T) {
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
	}

	for _, tc := range cases {
		t.Run(tc.Name, func(t *testing.T) {
			tc.Run(t, defaultCfg())
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
