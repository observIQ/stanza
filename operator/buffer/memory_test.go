package buffer

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMemoryBufferBuild(t *testing.T) {
	cfg := NewMemoryBufferConfig()
	operatorID := "operator"

	buffer, err := cfg.Build(operatorID)
	require.NoError(t, err)
	require.IsType(t, &MemoryBuffer{}, buffer)

	memBuffer := buffer.(*MemoryBuffer)
	assert.Equal(t, operatorID, memBuffer.operatorID)
	assert.Equal(t, cfg.MaxChunkDelay.Raw(), memBuffer.maxChunkDelay)
	assert.Equal(t, cfg.MaxChunkSize, memBuffer.maxChunkSize)
	assert.False(t, memBuffer.closed)
}
