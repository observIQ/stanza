// +build !windows

package windows

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/observiq/stanza/operator"
)

func TestWindowsOnly(t *testing.T) {
	require.False(t, operator.IsDefined("windows_eventlog_input"), "'windows_eventlog_input' should only be available on windows")
}
