package helper

import (
	"github.com/observiq/stanza/entry"
)

// NewLabelerConfig creates a new labeler config with default values
func NewLabelerConfig() LabelerConfig {
	return LabelerConfig{
		Attributes: make(map[string]ExprStringConfig),
	}
}

// LabelerConfig is the configuration of a labeler
type LabelerConfig struct {
	Attributes map[string]ExprStringConfig `json:"attributes" yaml:"attributes"`
}

// Build will build a labeler from the supplied configuration
func (c LabelerConfig) Build() (Labeler, error) {
	labeler := Labeler{
		attributes: make(map[string]*ExprString),
	}

	for k, v := range c.Attributes {
		exprString, err := v.Build()
		if err != nil {
			return labeler, err
		}

		labeler.attributes[k] = exprString
	}

	return labeler, nil
}

// Labeler is a helper that adds attributes to an entry
type Labeler struct {
	attributes map[string]*ExprString
}

// Label will add attributes to an entry
func (l *Labeler) Label(e *entry.Entry) error {
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
		e.AddLabel(k, rendered)
	}

	return nil
}
