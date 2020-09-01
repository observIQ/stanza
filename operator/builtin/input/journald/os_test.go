// +build !linux

package input

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/observiq/stanza/operator"
)

func TestLinuxOnly(t *testing.T) {
	require.False(t, operator.IsDefined("journald_input"), "'journald_input' should only be available on linux")
}
