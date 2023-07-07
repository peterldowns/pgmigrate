package shared

import "fmt"

// These will be set at build time with ldflags, see Justfile for how they're
// defined and passed.
var (
	Version = "unknown" //nolint:gochecknoglobals
	Commit  = "unknown" //nolint:gochecknoglobals
)

func VersionString() string {
	return fmt.Sprintf("%s+commit.%s", Version, Commit)
}
