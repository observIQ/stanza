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

package operatortest

import (
	"fmt"
	"io/ioutil"
	"path"
	"testing"

	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v2"
)

// ConfigUnmarshalTest is used for testing golden configs
type ConfigUnmarshalTest struct {
	Name      string
	Expect    interface{}
	ExpectErr bool
}

func configFromFileViaYaml(file string, config interface{}) error {
	bytes, err := ioutil.ReadFile(file)
	if err != nil {
		return fmt.Errorf("could not find config file: %s", err)
	}
	if err := yaml.Unmarshal(bytes, config); err != nil {
		return fmt.Errorf("failed to read config file as yaml: %s", err)
	}

	return nil
}

// Run Unmarshalls yaml files and compares them against the expected.
func (c ConfigUnmarshalTest) Run(t *testing.T, config interface{}) {
	yamlConfig := config
	yamlErr := configFromFileViaYaml(path.Join(".", "testdata", fmt.Sprintf("%s.yaml", c.Name)), yamlConfig)

	if c.ExpectErr {
		require.Error(t, yamlErr)
	} else {
		require.NoError(t, yamlErr)
		require.Equal(t, c.Expect, yamlConfig)
	}
}
