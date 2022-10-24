package postgres

import (
	"sort"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMustParseMigrations(t *testing.T) {
	var migrations []*migration
	assert.NotPanics(t, func() { migrations = mustParseMigrations() })

	assert.NotEmpty(t, migrations)

	assert.Truef(t,
		sort.SliceIsSorted(migrations, func(i, j int) bool {
			return migrations[i].version < migrations[j].version
		}),
		"returned migrations are not sorted ascending by version: %+v", migrations,
	)
}

func TestMigrationsFromVer(t *testing.T) {
	migrations := []*migration{
		{
			version: 0,
		},
		{
			version: 1,
		},
		{
			version: 5,
		},
		{
			version: 7,
		},
	}

	t.Run("0", func(t *testing.T) {
		assert.ElementsMatch(t,
			migrations[1:],
			migrationsFromVer(0, migrations),
		)
	})

	t.Run("5", func(t *testing.T) {
		assert.ElementsMatch(t,
			[]*migration{{version: 7}},
			migrationsFromVer(5, migrations),
		)
	})

	t.Run("7", func(t *testing.T) {
		assert.Empty(t, migrationsFromVer(7, migrations))
	})

	t.Run("8", func(t *testing.T) {
		assert.Empty(t, migrationsFromVer(8, migrations))
	})
}
