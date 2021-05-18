package version

import "fmt"

var (
	GitCommit string
	GitTag    string
)

// GetVersion returns the version of the stanza library
func GetVersion() string {
	if GitTag != "" {
		return GitTag
	}

	if GitCommit != "" {
		return fmt.Sprintf("git-commit: %s", GitCommit)
	}

	return "unknown"
}
