package entry

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestStringer(t *testing.T) {
	require.Equal(t, "default", Default.String())
	require.Equal(t, "trace", Trace.String())
	require.Equal(t, "trace2", Trace2.String())
	require.Equal(t, "trace3", Trace3.String())
	require.Equal(t, "trace4", Trace4.String())
	require.Equal(t, "debug", Debug.String())
	require.Equal(t, "debug2", Debug2.String())
	require.Equal(t, "debug3", Debug3.String())
	require.Equal(t, "debug4", Debug4.String())
	require.Equal(t, "info", Info.String())
	require.Equal(t, "info2", Info2.String())
	require.Equal(t, "info3", Info3.String())
	require.Equal(t, "info4", Info4.String())
	require.Equal(t, "notice", Notice.String())
	require.Equal(t, "warning", Warning.String())
	require.Equal(t, "warning2", Warning2.String())
	require.Equal(t, "warning3", Warning3.String())
	require.Equal(t, "warning4", Warning4.String())
	require.Equal(t, "error", Error.String())
	require.Equal(t, "error2", Error2.String())
	require.Equal(t, "error3", Error3.String())
	require.Equal(t, "error4", Error4.String())
	require.Equal(t, "critical", Critical.String())
	require.Equal(t, "alert", Alert.String())
	require.Equal(t, "emergency", Emergency.String())
	require.Equal(t, "emergency2", Emergency2.String())
	require.Equal(t, "emergency3", Emergency3.String())
	require.Equal(t, "emergency4", Emergency4.String())
	require.Equal(t, "catastrophe", Catastrophe.String())
	require.Equal(t, "19", Severity(19).String())
}
