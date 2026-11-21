package version

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSemVerFormatting(t *testing.T) {
	tests := []struct {
		name      string
		v         SemVer
		wantShort string
		wantStr   string
	}{
		{"plain", SemVer{Major: 1, Minor: 2, Patch: 3}, "1.2.3", "1.2.3"},
		{"appendix", SemVer{Major: 1, Minor: 2, Patch: 3, Appendix: "rc1"}, "1.2.3-rc1", "1.2.3-rc1"},
		{"commit", SemVer{Major: 1, GitCommit: "abc"}, "1.0.0", "1.0.0 (abc)"},
		{"appendix and commit", SemVer{Major: 5, Minor: 4, Patch: 4, Appendix: "x", GitCommit: "abc"}, "5.4.4-x", "5.4.4-x (abc)"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.wantShort, tc.v.Short())
			assert.Equal(t, tc.wantStr, tc.v.String())
		})
	}
}

func TestNewTrimsWhitespace(t *testing.T) {
	v, err := New("  1.2.3\n")
	require.NoError(t, err)

	assert.Equal(t, 1, v.Major)
	assert.Equal(t, 2, v.Minor)
	assert.Equal(t, 3, v.Patch)
}

func TestLoadPackageVarsParsesEmbeddedVersion(t *testing.T) {
	oldVer, oldCommit := CurSemVer, GitCommit
	t.Cleanup(func() { CurSemVer, GitCommit = oldVer, oldCommit })

	GitCommit = "deadbeef"
	CurSemVer = SemVer{}

	require.NoError(t, LoadPackageVars())

	assert.NotZero(t, CurSemVer.Major+CurSemVer.Minor+CurSemVer.Patch, "embedded ver file must hold a non-zero version")
	assert.Equal(t, "deadbeef", CurSemVer.GitCommit)
}
