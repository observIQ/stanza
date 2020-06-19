package plugin

import (
	"testing"

	"github.com/bluemedora/bplogagent/internal/testutil"
	"go.uber.org/zap/zaptest"
)

func NewTestBuildContext(t *testing.T) BuildContext {
	return BuildContext{
		Database: testutil.NewTestDatabase(t),
		Logger:   zaptest.NewLogger(t).Sugar(),
	}
}
