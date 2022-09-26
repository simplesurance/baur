//go:build dbtest
// +build dbtest

package postgres

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIsCompatible_AfterInit(t *testing.T) {
	client, cleanupFn := newTestClient(t)
	defer cleanupFn()

	require.NoError(t, client.Init(ctx))
	require.NoError(t, client.IsCompatible(ctx))
}

func TestIsCompatible_SchemaNotExist(t *testing.T) {
	client, cleanupFn := newTestClient(t)
	defer cleanupFn()

	err := client.IsCompatible(ctx)
	require.EqualError(t, err, "database schema does not exist")
}

func TestIsCompatible_OldBaurSchemaExist(t *testing.T) {
	client, cleanupFn := newTestClient(t)
	defer cleanupFn()

	_, err := client.db.Exec(ctx, "CREATE TABLE input_build();")
	require.NoError(t, err)

	err = client.IsCompatible(ctx)
	require.Error(t, err)
	require.Contains(t, err.Error(), "incompatible database schema")
}

func TestIsCompatible_SchemaVersionDoesNotMatch(t *testing.T) {
	client, cleanupFn := newTestClient(t)
	defer cleanupFn()

	require.NoError(t, client.Init(ctx))

	_, err := client.db.Exec(ctx, "UPDATE migrations set schema_version = 100")
	require.NoError(t, err)

	err = client.IsCompatible(ctx)
	assert.Error(t, err, "database schema version is not compatible")
}

func TestApplyMigrations(t *testing.T) {
	client, cleanupFn := newTestClient(t)
	defer cleanupFn()

	require.NoError(t, client.Init(ctx))

	err := client.ApplyMigrations(ctx, []*migration{
		{
			version: 1,
			sql:     "CREATE table t1()",
		},
		{
			version: 2,
			sql:     "CREATE table t2()",
		},
	})
	require.NoError(t, err)

	exist, err := client.tableExists(ctx, "t1")
	require.NoError(t, err)
	require.True(t, exist, "t1 table does not exist")

	exist, err = client.tableExists(ctx, "t2")
	require.NoError(t, err)
	require.True(t, exist, "t2 table does not exist")
}
