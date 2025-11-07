//go:build dbtest

package postgres

import (
	"context"
	"os"
	"testing"

	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/stretchr/testify/require"

	"github.com/simplesurance/baur/v5/internal/testutils/dbtest"
)

var ctx = context.Background()

func TestMain(m *testing.M) {
	os.Exit(m.Run())
}

func newTestClient(t *testing.T) (*Client, func()) {
	t.Helper()

	con, err := pgxpool.Connect(ctx, dbtest.PSQLURL())
	require.NoError(t, err)

	tx, err := con.Begin(ctx)
	require.NoError(t, err)

	client := Client{
		db:   tx,
		pool: con,
	}

	return &client, func() {
		err := tx.Rollback(ctx)
		require.NoError(t, err)

		con.Close()
	}
}
