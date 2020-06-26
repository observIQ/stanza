package entry

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestStringer(t *testing.T) {
	require.Equal(t, "default", Default.String())
	require.Equal(t, "trace", Trace.String())
	require.Equal(t, "debug", Debug.String())
	require.Equal(t, "info", Info.String())
	require.Equal(t, "notice", Notice.String())
	require.Equal(t, "warning", Warning.String())
	require.Equal(t, "error", Error.String())
	require.Equal(t, "critical", Critical.String())
	require.Equal(t, "alert", Alert.String())
	require.Equal(t, "emergency", Emergency.String())
	require.Equal(t, "catastrophe", Catastrophe.String())
	require.Equal(t, "12", Severity(12).String())
}
