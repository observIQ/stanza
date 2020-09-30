package operator

import (
	"fmt"
	"strings"

	"github.com/observiq/stanza/database"
	"go.uber.org/zap"
)

// BuildContext supplies contextual resources when building an operator.
type BuildContext struct {
	Database         database.Database
	Parameters       map[string]interface{}
	Logger           *zap.SugaredLogger
	Namespace        string
	DefaultOutputIDs []string
}

func (bc BuildContext) PrependNamespace(id string) string {
	if strings.HasPrefix(id, "$.") {
		return id
	}
	return fmt.Sprintf("%s.%s", bc.Namespace, id)
}

func (bc BuildContext) WithSubNamespace(namespace string) BuildContext {
	newBuildContext := bc.Copy()
	newBuildContext.Namespace = bc.PrependNamespace(namespace)
	return newBuildContext
}

func (bc BuildContext) WithDefaultOutputIDs(ids []string) BuildContext {
	newBuildContext := bc.Copy()
	newBuildContext.DefaultOutputIDs = ids
	return newBuildContext
}

func (bc BuildContext) Copy() BuildContext {
	return BuildContext{
		Database:         bc.Database,
		Parameters:       bc.Parameters,
		Logger:           bc.Logger,
		Namespace:        bc.Namespace,
		DefaultOutputIDs: bc.DefaultOutputIDs,
	}
}
