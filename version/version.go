package version

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
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

func init() {
	s, err := SemVerFromString(Version)
	if err == nil {
		CurSemVer = *s
	}

	CurSemVer.GitCommit = GitCommit
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

// SemVerFromString returns the SemVer representation of a string
func SemVerFromString(ver string) (*SemVer, error) {
	var appendix string

	spl := strings.Split(ver, ".")
	if len(spl) < 3 {
		return nil, errors.New("invalid format, should be <Major>.<Minor>.<Patch>[-appendix]")
	}

	major, err := strconv.ParseInt(spl[0], 10, 32)
	if err != nil {
		return nil, errors.New("could not convert major nr to int")
	}

	minor, err := strconv.ParseInt(spl[1], 10, 32)
	if err != nil {
		return nil, errors.New("could not convert minor nr to int")
	}

	patch, err := strconv.ParseInt(spl[2], 10, 32)
	if err != nil {
		return nil, errors.New("could not convert patch nr to int")
	}

	spl = strings.SplitN(ver, "-", 2)
	if len(spl) > 1 {
		appendix = spl[1]
	}

	return &SemVer{
		Major:    int(major),
		Minor:    int(minor),
		Patch:    int(patch),
		Appendix: appendix,
	}, nil
}
