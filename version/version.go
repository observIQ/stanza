package version

import "fmt"

var (
	// GitCommit set externally of the git commit this was built on
	GitCommit string

	// GitTag set externally of the git tag this was built on
	GitTag string
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
