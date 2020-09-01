package version

// Version is the current version of the stanza library
var Version string

// GitHash is the current git hash of the stanza library
var GitHash string

// GetVersion returns the version of the stanza library
func GetVersion() string {
	if Version != "" {
		return Version
	} else if GitHash != "" {
		return GitHash
	}
	return "unknown"
}
