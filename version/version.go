package version

import "fmt"

var (
	// GitCommit contains the git commit of this version.
	// It's set by make.
	GitCommit = ""

	// Version contains a semantic version number, must follow // https://semver.org/.
	// It's set by make.
	Version = ""
)

// FullVerNr returns a string containing Version and GitDescribe
func FullVerNr() string {
	return fmt.Sprintf("%s (%s)", Version, GitCommit)
}
