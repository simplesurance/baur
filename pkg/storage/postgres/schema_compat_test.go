//go:build dbtest

package postgres

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// If the schema version can not be determined, the schema must not be reported
// as compatible.
func TestIsCompatibleFailsIfSchemaVersionCanNotBeRead(t *testing.T) {
	clt, cleanup := newTestClient(t)
	defer cleanup()
	require.NoError(t, clt.Init(ctx))
	require.NoError(t, clt.IsCompatible(ctx), "precondition: a fresh database is compatible")

	_, err := clt.db.Exec(ctx, "INSERT INTO migrations (schema_version) VALUES (1)")
	require.NoError(t, err)

	err = clt.IsCompatible(ctx)

	require.ErrorContains(t, err, "migrations table contains >1 rows")
}
