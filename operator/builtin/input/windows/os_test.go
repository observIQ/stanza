//go:build !windows
// +build !windows

package windows

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/observiq/stanza/v2/operator"
)

func TestWindowsOnly(t *testing.T) {
	_, ok := operator.Lookup("windows_eventlog_input")
	require.False(t, ok, "'windows_eventlog_input' should only be available on windows")
}
