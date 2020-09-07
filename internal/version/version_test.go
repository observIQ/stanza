package version

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func resetVersion() {
	Version = ""
	GitHash = ""
}

func TestGetVersionWithVersion(t *testing.T) {
	Version = "0.1.1"
	defer resetVersion()
	require.Equal(t, Version, GetVersion())
}

func TestGetVersionWithGitHash(t *testing.T) {
	GitHash = "git hash"
	defer resetVersion()
	require.Equal(t, GitHash, GetVersion())
}

func TestGetVersionWithUnknownVersion(t *testing.T) {
	defer resetVersion()
	require.Equal(t, "unknown", GetVersion())
}
