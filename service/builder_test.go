package service

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func TestBuilder(t *testing.T) {
	logger := zap.New(zapcore.NewNopCore()).Sugar()
	configFile, pluginDir, databaseFile := "config", "plugin", "database"
	testCases := []struct {
		desc      string
		buildFunc func(*AgentServiceBuilder)
		expected  *AgentServiceBuilder
	}{
		{
			desc:      "No options",
			buildFunc: func(*AgentServiceBuilder) {},
			expected:  &AgentServiceBuilder{},
		},
		{
			desc: "With Pluging Dir",
			buildFunc: func(b *AgentServiceBuilder) {
				b.WithPluginDir(pluginDir)
			},
			expected: &AgentServiceBuilder{
				pluginDir: &pluginDir,
			},
		},
		{
			desc: "With Logger",
			buildFunc: func(b *AgentServiceBuilder) {
				b.WithLogger(logger)
			},
			expected: &AgentServiceBuilder{
				logger: logger,
			},
		},
		{
			desc: "With Config File",
			buildFunc: func(b *AgentServiceBuilder) {
				b.WithConfigFile(configFile)
			},
			expected: &AgentServiceBuilder{
				configFile: &configFile,
			},
		},
		{
			desc: "With Database File",
			buildFunc: func(b *AgentServiceBuilder) {
				b.WithDatabaseFile(databaseFile)
			},
			expected: &AgentServiceBuilder{
				databaseFile: &databaseFile,
			},
		},
		{
			desc: "With All",
			buildFunc: func(b *AgentServiceBuilder) {
				b.WithPluginDir(pluginDir)
				b.WithLogger(logger)
				b.WithConfigFile(configFile)
				b.WithDatabaseFile(databaseFile)
			},
			expected: &AgentServiceBuilder{
				pluginDir:    &pluginDir,
				logger:       logger,
				configFile:   &configFile,
				databaseFile: &databaseFile,
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			builder := NewBuilder()
			tc.buildFunc(builder)
			assert.Equal(t, tc.expected, builder)
		})
	}
}
