package version

import "fmt"

var (
	// GitCommit contains the git commit of this version.
	// It's set by make.
	GitCommit = ""

	// Version contains a semantic version number, must follow // https://semver.org/.
	// It's set by make.
	Version = ""

	// Appendix is appended after a hypen to the version number. It can be
	// used to mark a build as a prerelease.
	Appendix = ""
)

// FullVerNr returns a string containing Version and GitDescribe
func FullVerNr() string {
	if Appendix == "" {
		return fmt.Sprintf("%s (%s)", Version, GitCommit)
	}

	return fmt.Sprintf("%s-%s (%s)", Version, Appendix, GitCommit)
}
