package postgres

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/simplesurance/baur/v5/pkg/storage"
)

func TestStrArgList(t *testing.T) {
	assert.Equal(t, "[]", strArgList())
	assert.Equal(t, "['a']", strArgList("a"))
	assert.Equal(t, "['a', '1', 'true']", strArgList("a", 1, true))
}

func TestQueryValueStr(t *testing.T) {
	tests := []struct {
		pairs, perPair int
		want           string
	}{
		{0, 3, ""},
		{1, 1, "($1)"},
		{1, 3, "($1, $2, $3)"},
		{2, 2, "($1, $2), ($3, $4)"},
		{3, 2, "($1, $2), ($3, $4), ($5, $6)"},
	}

	for _, tc := range tests {
		// normWS: pairs are separated by ",  " (two spaces)
		assert.Equal(t, tc.want, normWS(queryValueStr(tc.pairs, tc.perPair)),
			"pairs=%d perPair=%d", tc.pairs, tc.perPair)
	}
}

func TestQueryValuePairFirstConstStr(t *testing.T) {
	assert.Empty(t, queryValuePairFirstConstStr(0))
	assert.Equal(t, "($1, $2)", queryValuePairFirstConstStr(1))
	assert.Equal(t, "($1, $2), ($1, $3), ($1, $4)", queryValuePairFirstConstStr(3))
}

func TestClonedSortedInputFiles(t *testing.T) {
	in := []*storage.InputFile{
		{Path: "b", Digest: "1"},
		{Path: "a", Digest: "2"},
		{Path: "a", Digest: "1"},
	}
	orig := append([]*storage.InputFile{}, in...)

	got := clonedSortedInputfiles(in)

	assert.Equal(t, []*storage.InputFile{
		{Path: "a", Digest: "1"},
		{Path: "a", Digest: "2"},
		{Path: "b", Digest: "1"},
	}, got)
	assert.Equal(t, orig, in, "input slice must not be modified")
}

func TestClonedSortedInputStrings(t *testing.T) {
	in := []*storage.InputString{
		{String: "b", Digest: "1"},
		{String: "a", Digest: "2"},
		{String: "a", Digest: "1"},
	}
	orig := append([]*storage.InputString{}, in...)

	got := clonedSortedInputStrings(in)

	assert.Equal(t, []*storage.InputString{
		{String: "a", Digest: "1"},
		{String: "a", Digest: "2"},
		{String: "b", Digest: "1"},
	}, got)
	assert.Equal(t, orig, in, "input slice must not be modified")
}

func TestClonedSortedInputEnvVars(t *testing.T) {
	in := []*storage.InputEnvVar{
		{Name: "B", Digest: "1"},
		{Name: "A", Digest: "2"},
		{Name: "A", Digest: "1"},
	}
	orig := append([]*storage.InputEnvVar{}, in...)

	got := clonedSortedInputEnvVars(in)

	assert.Equal(t, []*storage.InputEnvVar{
		{Name: "A", Digest: "1"},
		{Name: "A", Digest: "2"},
		{Name: "B", Digest: "1"},
	}, got)
	assert.Equal(t, orig, in, "input slice must not be modified")
}

func TestClonedSortedInputTasks(t *testing.T) {
	in := []*storage.InputTaskInfo{
		{Name: "b", Digest: "1"},
		{Name: "a", Digest: "2"},
		{Name: "a", Digest: "1"},
	}
	orig := append([]*storage.InputTaskInfo{}, in...)

	got := clonedSortedInputTasks(in)

	assert.Equal(t, []*storage.InputTaskInfo{
		{Name: "a", Digest: "1"},
		{Name: "a", Digest: "2"},
		{Name: "b", Digest: "1"},
	}, got)
	assert.Equal(t, orig, in, "input slice must not be modified")
}

func TestQueryError(t *testing.T) {
	cause := errors.New("boom")
	err := newQueryError("\n  SELECT 1\n", cause, "a", 2)

	require.ErrorIs(t, err, cause)
	assert.Equal(t, "boom\nquery:\n---\nSELECT 1\n---\narguments: ['a', '2']", err.Error())
}

func TestRedactedURL(t *testing.T) {
	got, err := redactedURL("postgres://user:secret@host:5432/db?sslmode=disable")
	require.NoError(t, err)
	assert.Equal(t, "postgres://user:xxxxx@host:5432/db?sslmode=disable", got)

	got, err = redactedURL("postgres://user@host/db")
	require.NoError(t, err)
	assert.Equal(t, "postgres://user@host/db", got)

	_, err = redactedURL("postgres://user:pa ss@host:notaport/db")
	require.Error(t, err)
}

func TestSchemaVersionBounds(t *testing.T) {
	migrations := mustParseMigrations()
	require.NotEmpty(t, migrations)

	latest := migrations[len(migrations)-1].version

	assert.Equal(t, maxSchemaVer, latest,
		"maxSchemaVer must be the version of the newest migration")
	assert.Equal(t, maxSchemaVer, (&Client{}).MaxSchemaVersion())
	assert.LessOrEqual(t, minSchemaVer, maxSchemaVer)

	for i, m := range migrations {
		assert.NotEmpty(t, m.sql, "migration %d is empty", m.version)

		if i > 0 {
			assert.Equal(t, migrations[i-1].version+1, m.version,
				"migration versions must be consecutive")
		}
	}
}
