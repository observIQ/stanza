package service

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBuilder(t *testing.T) {
	config := &Config{}
	databaseFile := "database"
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
			desc: "With Config File",
			buildFunc: func(b *AgentServiceBuilder) {
				b.WithConfig(config)
			},
			expected: &AgentServiceBuilder{
				config: config,
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
				b.WithConfig(config)
				b.WithDatabaseFile(databaseFile)
			},
			expected: &AgentServiceBuilder{
				config:       config,
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
