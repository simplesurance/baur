// +build dbtest

package postgres

import (
	"context"
	"os"
	"testing"

	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var dbURL = "postgres://postgres@localhost:5434/baur?sslmode=disable&connect_timeout=5"

var ctx = context.Background()

func TestMain(m *testing.M) {
	if url := os.Getenv("BAUR_POSTGRESQL_URL"); url != "" {
		dbURL = url
	}

	os.Exit(m.Run())
}

func newTestClient(t *testing.T) (*Client, func()) {
	t.Helper()

	con, err := pgxpool.Connect(ctx, dbURL)
	require.NoError(t, err)

	tx, err := con.Begin(ctx)
	require.NoError(t, err)

	client := Client{
		db:   tx,
		pool: con,
	}

	return &client, func() {
		err := tx.Rollback(ctx)
		assert.NoError(t, err)

		con.Close()
	}
}
