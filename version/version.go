package version

import "fmt"

var (
	// GitDescribe contains the git describe string of this version.
	// It's set by make.
	GitDescribe = ""

	// Version contains a semantic version number, must follow https://semver.org/
	Version = "0.0.0"
)

func FullVerNr() string {
	return fmt.Sprintf("%s (git ref: %s)", Version, GitDescribe)
}
