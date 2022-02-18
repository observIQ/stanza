package operator

import (
	"fmt"
	"strings"

	"github.com/observiq/stanza/database"
	"github.com/observiq/stanza/logger"
	"go.uber.org/zap"
)

// BuildContext supplies contextual resources when building an operator.
type BuildContext struct {
	Database         database.Database
	Parameters       map[string]interface{}
	Logger           *logger.Logger
	Namespace        string
	DefaultOutputIDs []string
	PluginDepth      int
}

// PrependNamespace adds the current namespace of the build context to the
// front of the given ID if that ID is not already namespaced up to the root level
func (bc BuildContext) PrependNamespace(id string) string {
	if strings.HasPrefix(id, "$.") {
		return id
	}
	return fmt.Sprintf("%s.%s", bc.Namespace, id)
}

// WithSubNamespace creates a new build context with a more specific namespace
func (bc BuildContext) WithSubNamespace(namespace string) BuildContext {
	newBuildContext := bc.Copy()
	newBuildContext.Namespace = bc.PrependNamespace(namespace)
	return newBuildContext
}

// WithDefaultOutputIDs sets the default output IDs for the current context or
// the current operator build
func (bc BuildContext) WithDefaultOutputIDs(ids []string) BuildContext {
	newBuildContext := bc.Copy()
	newBuildContext.DefaultOutputIDs = ids
	return newBuildContext
}

// WithIncrementedDepth returns a new build context with an incremented
// plugin depth
func (bc BuildContext) WithIncrementedDepth() BuildContext {
	newBuildContext := bc.Copy()
	newBuildContext.PluginDepth++
	return newBuildContext
}

// Copy creates a copy of the build context
func (bc BuildContext) Copy() BuildContext {
	return BuildContext{
		Database:         bc.Database,
		Parameters:       bc.Parameters,
		Logger:           bc.Logger,
		Namespace:        bc.Namespace,
		DefaultOutputIDs: bc.DefaultOutputIDs,
		PluginDepth:      bc.PluginDepth,
	}
}

// NewBuildContext creates a new build context with the given database, logger, and the
// default namespace
func NewBuildContext(db database.Database, lg *zap.SugaredLogger) BuildContext {
	return BuildContext{
		Database:         db,
		Parameters:       nil,
		Logger:           logger.New(lg),
		Namespace:        "$",
		DefaultOutputIDs: []string{},
	}
}
