package dbtest

import (
	"context"
	"net/url"
	"os"

	"github.com/jackc/pgx/v4"
)

const defPSQLURL = "postgres://postgres@localhost:5434/baur?sslmode=disable&connect_timeout=5"

// DDBURL returns the value of the environment variable BAUR_POSTGRESQL_URL.
// If the environment variable is undefined or empty, defDBURL is returned.
func PSQLURL() string {
	if url := os.Getenv("BAUR_POSTGRESQL_URL"); url != "" {
		return url
	}

	return defPSQLURL
}

// CreateDB creates a new database and returns the connection URL string of it.
// The database is created at the postgresql instance reachable via PSQLURL()
func CreateDB(name string) (string, error) {
	ctx := context.Background()
	psqlURL := PSQLURL()

	con, err := pgx.Connect(ctx, psqlURL)
	if err != nil {
		return "", err
	}

	defer con.Close(ctx)

	_, err = con.Query(ctx, "CREATE DATABASE "+name)
	if err != nil {
		return "", err
	}

	u, err := url.Parse(psqlURL)
	if err != nil {
		return "", err
	}

	u.Path = name

	return u.String(), nil
}
