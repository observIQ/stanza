package testutil

import (
	"testing"

	plugin "github.com/bluemedora/bplogagent/plugin"
	"go.uber.org/zap/zaptest"
)

func NewTestBuildContext(t *testing.T) plugin.BuildContext {
	return plugin.BuildContext{
		Database: nil,
		Logger:   zaptest.NewLogger(t).Sugar(),
	}
}
