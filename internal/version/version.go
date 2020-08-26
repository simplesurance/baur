package version

import (
	"fmt"
	"io"

	"github.com/pkg/errors"
)

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

	// CurSemVer is the current version
	CurSemVer = SemVer{}
)

// LoadPackageVars parses the package variable and sets CurSemVer
func LoadPackageVars() error {
	s, err := FromString(Version)
	if err != nil {
		return errors.Wrapf(err, "parsing version '%s' failed", Version)
	}

	CurSemVer = *s
	CurSemVer.GitCommit = GitCommit

	return nil
}

// SemVer holds a semantic version
type SemVer struct {
	Major     int
	Minor     int
	Patch     int
	Appendix  string
	GitCommit string
}

// String returns the string representation
func (s *SemVer) String() string {
	ver := s.Short()

	if s.GitCommit != "" {
		ver += fmt.Sprintf(" (%s)", s.GitCommit)
	}

	return ver
}

// Short returns the version without GitCommit
func (s *SemVer) Short() string {
	ver := fmt.Sprintf("%d.%d.%d", s.Major, s.Minor, s.Patch)

	if s.Appendix != "" {
		ver += "-" + s.Appendix
	}
	return ver
}

// FromString returns the SemVer representation of a string
func FromString(ver string) (*SemVer, error) {
	var appendix string
	var major, minor, patch int

	matches, err := fmt.Sscanf(ver, "%d.%d.%d-%s", &major, &minor, &patch, &appendix)
	if (err != nil && err != io.ErrUnexpectedEOF) || matches < 1 {
		return nil, errors.Wrapf(err, "invalid format, should be <Major>[.<Minor>[.<Patch>[-appendix]]]")
	}

	return &SemVer{
		Major:    major,
		Minor:    minor,
		Patch:    patch,
		Appendix: appendix,
	}, nil
}
