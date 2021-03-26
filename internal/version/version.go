package version

import (
	_ "embed" // is required to initialize the rawVersion variable with content from the ver file
	"fmt"
	"io"
	"strings"
)

// The GitCommit and Appendix variable are set via the go linker (go build -ldflags -X [...]).
var (
	// GitCommit contains the git commit of this version.
	GitCommit = ""

	// Appendix is appended after a hypen to the version string.
	Appendix = ""
)

// rawVersion contains a semantic version string (in https://semver.org
// format). It is initialized with the content of the ver file.
//go:embed ver
var rawVersion string

// CurSemVer is the current version
var CurSemVer = SemVer{}

// LoadPackageVars parses the rawVersion, GitCommit and Appendix package
// variables and initializes CurSemVer accordingly.
func LoadPackageVars() error {
	s, err := New(rawVersion)
	if err != nil {
		return fmt.Errorf("parsing version %q failed: %w", rawVersion, err)
	}

	CurSemVer = *s
	CurSemVer.GitCommit = GitCommit

	return nil
}

// SemVer represents a semantic version number.
type SemVer struct {
	Major     int
	Minor     int
	Patch     int
	Appendix  string
	GitCommit string
}

// String returns the version in MAJOR.MINOR.PATCH-APPENDIX (GITCOMMIT) format.
// If GitCommit is empty, the part is omitted.
func (s *SemVer) String() string {
	ver := s.Short()

	if s.GitCommit != "" {
		ver += fmt.Sprintf(" (%s)", s.GitCommit)
	}

	return ver
}

// Short returns the version in MAJOR.MINOR.PATCH-APPENDIX format.
func (s *SemVer) Short() string {
	ver := fmt.Sprintf("%d.%d.%d", s.Major, s.Minor, s.Patch)

	if s.Appendix != "" {
		ver += "-" + s.Appendix
	}
	return ver
}

// New returns a Semver that is initialized by parsing it from a string.
func New(ver string) (*SemVer, error) {
	var appendix string
	var major, minor, patch int

	ver = strings.TrimSpace(ver)
	matches, err := fmt.Sscanf(ver, "%d.%d.%d-%s", &major, &minor, &patch, &appendix)
	if (err != nil && err != io.ErrUnexpectedEOF) || matches < 1 {
		return nil, fmt.Errorf("invalid format, should be <Major>[.<Minor>[.<Patch>[-appendix]]]: %w", err)
	}

	return &SemVer{
		Major:    major,
		Minor:    minor,
		Patch:    patch,
		Appendix: appendix,
	}, nil
}
