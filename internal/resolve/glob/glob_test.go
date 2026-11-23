package glob

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/simplesurance/baur/v5/internal/testutils/fstest"
)

func TestResolve(t *testing.T) {
	dir := fstest.TempDir(t)
	for _, f := range []string{"a.go", "b.go", "c.txt", "sub/d.go", "sub/deep/e.go"} {
		fstest.WriteToFile(t, []byte("x"), filepath.Join(dir, f))
	}

	rel := func(paths []string) []string {
		res := make([]string, 0, len(paths))
		for _, p := range paths {
			r, err := filepath.Rel(dir, p)
			require.NoError(t, err)
			res = append(res, filepath.ToSlash(r))
		}

		return res
	}

	tests := []struct {
		name    string
		pattern string
		want    []string
	}{
		{"single star matches in one dir only", "*.go", []string{"a.go", "b.go"}},
		{"double star descends into subdirs", "**/*.go", []string{"a.go", "b.go", "sub/d.go", "sub/deep/e.go"}},
		{"exact file", "c.txt", []string{"c.txt"}},
		{"directories are not returned", "sub", []string{}},
		{"wildcard without match is not an error", "*.nothing", []string{}},
		{"star in dir part", "*/*.go", []string{"sub/d.go"}},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var r Resolver

			got, err := r.Resolve(filepath.Join(dir, tc.pattern))
			require.NoError(t, err)
			assert.ElementsMatch(t, tc.want, rel(got))
		})
	}
}

func TestResolveNonExistingPathReturnsErrNotExist(t *testing.T) {
	var r Resolver
	dir := fstest.TempDir(t)

	_, err := r.Resolve(filepath.Join(dir, "missing.txt"))
	require.ErrorIs(t, err, os.ErrNotExist)

	_, err = r.Resolve(filepath.Join(dir, "missingdir", "*.go"))
	require.ErrorIs(t, err, os.ErrNotExist)
}

func TestResolveErrorContainsPattern(t *testing.T) {
	var r Resolver

	_, err := r.Resolve("[")

	require.Error(t, err)
	assert.Contains(t, err.Error(), `resolving "[" failed`)
}

func TestMatches(t *testing.T) {
	var r Resolver

	t.Run("returns the first matching pattern", func(t *testing.T) {
		ok, pattern, err := r.Matches("a/b/c.go", []string{"*.txt", "**/*.go", "a/**"})

		require.NoError(t, err)
		assert.True(t, ok)
		assert.Equal(t, "**/*.go", pattern)
	})

	t.Run("no match", func(t *testing.T) {
		ok, pattern, err := r.Matches("a/b/c.go", []string{"*.go", "x/**"})

		require.NoError(t, err)
		assert.False(t, ok)
		assert.Empty(t, pattern)
	})

	t.Run("no patterns", func(t *testing.T) {
		ok, _, err := r.Matches("a", nil)

		require.NoError(t, err)
		assert.False(t, ok)
	})

	t.Run("single star does not cross directories", func(t *testing.T) {
		ok, _, err := r.Matches("a/b.go", []string{"*.go"})

		require.NoError(t, err)
		assert.False(t, ok)
	})

	t.Run("invalid pattern is an error", func(t *testing.T) {
		_, _, err := r.Matches("a", []string{"["})

		require.Error(t, err)
		assert.Contains(t, err.Error(), `pattern "["`)
	})
}
