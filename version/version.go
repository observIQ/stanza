package version

import "runtime/debug"

var version = func() string {
	bi, ok := debug.ReadBuildInfo()
	if !ok {
		return "unknown"
	}

	for _, mod := range bi.Deps {
		if mod.Path == "github.com/observiq/stanza" {
			return mod.Version
		}
	}
	return "unknown"
}()

// GetVersion returns the version of the stanza library
func GetVersion() string {
	return version
}
