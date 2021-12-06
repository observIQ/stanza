package service

import (
	"io/ioutil"
	"path"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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

func TestBuilder_validateNonGlobConfig(t *testing.T) {
	testCases := []struct {
		desc     string
		testFunc func(*testing.T)
	}{
		{
			desc: "Nil configFile pointer",
			testFunc: func(t *testing.T) {
				builder := NewBuilder()
				err := builder.validateNonGlobConfig()
				assert.NoError(t, err)
			},
		},
		{
			desc: "configFile bad pattern",
			testFunc: func(t *testing.T) {
				builder := NewBuilder().WithConfigFile(`[`)
				err := builder.validateNonGlobConfig()
				assert.ErrorIs(t, err, filepath.ErrBadPattern)
			},
		},
		{
			desc: "configFile glob pattern",
			testFunc: func(t *testing.T) {
				// Create mock up file system
				tmpDir := t.TempDir()
				for i := 0; i < 3; i++ {
					tmpFile, err := ioutil.TempFile(tmpDir, "config")
					require.NoError(t, err)
					// Close the file right away as we don't need it open
					tmpFile.Close()
				}

				builder := NewBuilder().WithConfigFile(path.Join(tmpDir, "config*"))
				err := builder.validateNonGlobConfig()
				assert.Error(t, err)
			},
		},
		{
			desc: "configFile single file",
			testFunc: func(t *testing.T) {
				// Create mock up file system
				tmpDir := t.TempDir()
				tmpFile, err := ioutil.TempFile(tmpDir, "config")
				require.NoError(t, err)
				// Close the file right away as we don't need it open
				tmpFile.Close()

				builder := NewBuilder().WithConfigFile(path.Join(tmpDir, tmpFile.Name()))
				err = builder.validateNonGlobConfig()
				assert.NoError(t, err)
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.desc, tc.testFunc)
	}
}
