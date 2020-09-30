package operator

import (
	"fmt"
	"plugin"

	"github.com/observiq/stanza/database"
	"github.com/observiq/stanza/plugin"
	"go.uber.org/zap"
)

// BuildContext supplies contextual resources when building an operator.
type BuildContext struct {
	Database       database.Database
	Parameters     map[string]interface{}
	Logger         *zap.SugaredLogger
	Namespace      string
	PluginRegistry plugin.Registry
}

func (b BuildContext) PrependNamespace(id string) string {
	return fmt.Sprintf("%s.%s", b.Namespace, id)
}

func (b BuildContext) WithSubNamespace(namespace string) BuildContext {
	return BuildContext{
		Database:   b.Database,
		Parameters: b.Parameters,
		Logger:     b.Logger,
		Namespace:  b.PrependNamespace(namespace),
	}
}
