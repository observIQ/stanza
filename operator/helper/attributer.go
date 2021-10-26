package helper

import (
	"github.com/open-telemetry/opentelemetry-log-collection/entry"
)

// NewAttributerConfig creates a new attributer config with default values
func NewAttributerConfig() AttributerConfig {
	return AttributerConfig{
		Attributes: make(map[string]ExprStringConfig),
	}
}

// AttributerConfig is the configuration of an attributer
type AttributerConfig struct {
	Attributes map[string]ExprStringConfig `json:"attributes" yaml:"attributes"`
}

// Build will build an attributer from the supplied configuration
func (c AttributerConfig) Build() (Attributer, error) {
	attributer := Attributer{
		attributes: make(map[string]*ExprString),
	}

	for k, v := range c.Attributes {
		exprString, err := v.Build()
		if err != nil {
			return attributer, err
		}

		attributer.attributes[k] = exprString
	}

	return attributer, nil
}

// Attributer is a helper that adds attributes to an entry
type Attributer struct {
	attributes map[string]*ExprString
}

// Attribute will add attributes to an entry
func (l *Attributer) Attribute(e *entry.Entry) error {
	if len(l.attributes) == 0 {
		return nil
	}

	env := GetExprEnv(e)
	defer PutExprEnv(env)

	for k, v := range l.attributes {
		rendered, err := v.Render(env)
		if err != nil {
			return err
		}
		e.AddAttribute(k, rendered)
	}

	return nil
}
