package helper

import (
	"github.com/observiq/stanza/entry"
)

// NewLabelerConfig creates a new labeler config with default values
func NewLabelerConfig() LabelerConfig {
	return LabelerConfig{
		Labels: make(map[string]ExprStringConfig),
	}
}

// LabelerConfig is the configuration of a labeler
type LabelerConfig struct {
	Labels map[string]ExprStringConfig `json:"labels" yaml:"labels"`
}

// Build will build a labeler from the supplied configuration
func (c LabelerConfig) Build() (Labeler, error) {
	labeler := Labeler{
		labels: make(map[string]*ExprString),
	}

	for k, v := range c.Labels {
		exprString, err := v.Build()
		if err != nil {
			return labeler, err
		}

		labeler.labels[k] = exprString
	}

	return labeler, nil
}

// Labeler is a helper that adds labels to an entry
type Labeler struct {
	labels map[string]*ExprString
}

// Label will add labels to an entry
func (l *Labeler) Label(e *entry.Entry) error {
	if len(l.labels) == 0 {
		return nil
	}

	env := GetExprEnv(e)
	defer PutExprEnv(env)

	for k, v := range l.labels {
		rendered, err := v.Render(env)
		if err != nil {
			return err
		}
		e.AddLabel(k, rendered)
	}

	return nil
}
