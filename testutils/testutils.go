package testutils

import (
	"os"
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
