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

package retain

import (
	"context"
	"fmt"
	"strings"

	"github.com/observiq/stanza/v2/entry"
	"github.com/observiq/stanza/v2/operator"
	"github.com/observiq/stanza/v2/operator/helper"
)

func init() {
	operator.Register("retain", func() operator.Builder { return NewRetainOperatorConfig("") })
}

// NewRetainOperatorConfig creates a new retain operator config with default values
func NewRetainOperatorConfig(operatorID string) *RetainOperatorConfig {
	return &RetainOperatorConfig{
		TransformerConfig: helper.NewTransformerConfig(operatorID, "retain"),
	}
}

// RetainOperatorConfig is the configuration of a retain operator
type RetainOperatorConfig struct {
	helper.TransformerConfig `mapstructure:",squash" yaml:",inline"`
	Fields                   []entry.Field `mapstructure:"fields" json:"fields" yaml:"fields"`
}

// Build will build a retain operator from the supplied configuration
func (c RetainOperatorConfig) Build(context operator.BuildContext) ([]operator.Operator, error) {
	transformerOperator, err := c.TransformerConfig.Build(context)
	if err != nil {
		return nil, err
	}
	if c.Fields == nil || len(c.Fields) == 0 {
		return nil, fmt.Errorf("retain: 'fields' is empty")
	}

	retainOp := &RetainOperator{
		TransformerOperator: transformerOperator,
		Fields:              c.Fields,
	}

	for _, field := range c.Fields {
		typeCheck := field.String()
		if strings.HasPrefix(typeCheck, "$resource") {
			retainOp.AllResourceFields = true
			continue
		}
		if strings.HasPrefix(typeCheck, "$labels") {
			retainOp.AllAttributeFields = true
			continue
		}
		retainOp.AllBodyFields = true
	}
	return []operator.Operator{retainOp}, nil
}

// RetainOperator keeps the given fields and deletes the rest.
type RetainOperator struct {
	helper.TransformerOperator
	Fields             []entry.Field
	AllBodyFields      bool
	AllAttributeFields bool
	AllResourceFields  bool
}

// Process will process an entry with a retain transformation.
func (p *RetainOperator) Process(ctx context.Context, entry *entry.Entry) error {
	return p.ProcessWith(ctx, entry, p.Transform)
}

// Transform will apply the retain operation to an entry
func (p *RetainOperator) Transform(e *entry.Entry) error {
	newEntry := entry.New()
	newEntry.Timestamp = e.Timestamp

	if !p.AllResourceFields {
		newEntry.Resource = e.Resource
	}
	if !p.AllAttributeFields {
		newEntry.Labels = e.Labels
	}
	if !p.AllBodyFields {
		newEntry.Record = e.Record
	}

	for _, field := range p.Fields {
		val, ok := e.Get(field)
		if !ok {
			continue
		}
		err := newEntry.Set(field, val)
		if err != nil {
			return err
		}
	}

	*e = *newEntry
	return nil
}
