package postgres

import (
	"context"
	"embed"
	"errors"
	"fmt"
	"path"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/jackc/pgx/v4"

	"github.com/simplesurance/baur/v5/pkg/storage"
)

const (
	// minSchemaVer is the minimum required database schema version
	minSchemaVer int32 = 4
	// maxSchemaVer is the highest database schema version that is compatible
	maxSchemaVer int32 = 5
)

// migration represents a database schema migration.
type migration struct {
	version int32
	sql     string
}

//go:embed migrations/*
var migrationFs embed.FS

// mustParseMigrations reads the sql schema migrations from migrationFs and
// returns them sorted ascending by version.
func mustParseMigrations() []*migration {
	// this function is normally only run once per baur invocation
	const panicMsgPrefix = "postgres: migrations: "
	const baseDir = "migrations"
	validFilenameRe := regexp.MustCompile(`^[0-9]+.sql$`)
	var res []*migration //nolint:prealloc

	entries, err := migrationFs.ReadDir(baseDir)
	if err != nil {
		panic(panicMsgPrefix + err.Error())
	}

	for _, e := range entries {
		name := e.Name()
		// use path.Join instead of filepath.Join because on embed.FS
		// the directory separator is always `/` independent of the OS
		path := path.Join(baseDir, name)
		if !e.Type().IsRegular() {
			panic(fmt.Sprintf(panicMsgPrefix+"%q is not a regular file", path))
		}

		if !validFilenameRe.MatchString(name) {
			panic(fmt.Sprintf(
				panicMsgPrefix+"%q invalid filename, expecting only migration files matching regex: %q",
				name, validFilenameRe.String(),
			))
		}

		content, err := migrationFs.ReadFile(path)
		if err != nil {
			panic(panicMsgPrefix + err.Error())
		}
		ver, err := strconv.ParseInt(strings.TrimRight(name, ".sql"), 10, 32)
		if err != nil {
			panic(panicMsgPrefix + "could not parse numeric version: " + err.Error())
		}

		if ver < 0 {
			panic(fmt.Sprintf(
				panicMsgPrefix+"%q has schema version %d, expecting version >=1",
				path, ver,
			))
		}

		res = append(res, &migration{
			sql:     string(content),
			version: int32(ver),
		})
	}

	sort.Slice(res, func(i, j int) bool {
		return res[i].version < res[j].version
	})

	return res
}

// Init creates the baur tables in the postgresql database.
// If the database already exist, storage.ErrExist is returned.
func (c *Client) Init(ctx context.Context) error {
	err := c.schemaExist(ctx)
	if err == nil {
		return storage.ErrExists
	}

	if !errors.Is(err, storage.ErrNotExist) {
		return err
	}

	return c.applyMigrations(ctx, mustParseMigrations())
}

// migrationsFromVer returns a slice from migrations that only contains
// migrations with a version > minVer.
// if no migration has a version > minver, nil is returned.
func migrationsFromVer(minVer int32, migrations []*migration) []*migration {
	for i, m := range migrations {
		if m.version > minVer {
			return migrations[i:]
		}
	}

	return nil
}

// Upgrade transitions the database schema to the current version by running
// all migrations sql script that have a newer version then current schema
// version that the database uses.
// If the database does not exist, storage.ErrNotExist is returned.
func (c *Client) Upgrade(ctx context.Context) error {
	err := c.schemaExist(ctx)
	if err != nil {
		if errors.Is(err, storage.ErrNotExist) {
			return c.Init(ctx)
		}

		return err
	}

	if err := c.v0SchemaNotExits(ctx); err != nil {
		return err
	}

	ver, err := c.schemaVersion(ctx)
	if err != nil {
		return err
	}

	migrations := migrationsFromVer(ver, mustParseMigrations())
	return c.applyMigrations(ctx, migrations)
}

func (c *Client) applyMigrations(ctx context.Context, migrations []*migration) error {
	return c.db.BeginFunc(ctx, func(tx pgx.Tx) error {
		for _, m := range migrations {
			_, err := tx.Exec(ctx, m.sql)
			if err != nil {
				return fmt.Errorf("applying database schema migration %d failed: %w", m.version, err)
			}
		}

		err := setSchemaVersion(ctx, tx, migrations[len(migrations)-1].version)
		if err != nil {
			return fmt.Errorf("updating schema version failed: %w", err)
		}

		return nil
	})
}

func setSchemaVersion(ctx context.Context, tx pgx.Tx, ver int32) error {
	_, err := tx.Exec(ctx, "UPDATE migrations SET schema_version=$1", ver)
	return err
}

// IsCompatible checks if the database schema exist and has the required
// migration version.
func (c *Client) IsCompatible(ctx context.Context) error {
	if err := c.v0SchemaNotExits(ctx); err != nil {
		return err
	}

	if err := c.schemaExist(ctx); err != nil {
		return err
	}

	return c.ensureSchemaIsCompatible(ctx)
}

// schemaVersion returns the version of the current schema in the database.
func (c *Client) schemaVersion(ctx context.Context) (int32, error) {
	var rowsCount int
	var ver int32

	rows, err := c.db.Query(ctx, "SELECT schema_version from migrations")
	if err != nil {
		return -1, fmt.Errorf("querying schema_version failed: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		if rowsCount != 0 {
			return -1, errors.New("migrations table contains >1 rows")
		}

		err = rows.Scan(&ver)
		if err != nil {
			return -1, err
		}

		rowsCount++
	}

	if err := rows.Err(); err != nil {
		return -1, err
	}

	if rowsCount != 1 {
		return -1, fmt.Errorf("read %d rows from migrations table, expected 1", rowsCount)
	}

	return ver, nil
}

func (c *Client) ensureSchemaIsCompatible(ctx context.Context) error {
	ver, err := c.schemaVersion(ctx)
	if err != nil {
		return nil
	}

	if ver < minSchemaVer || ver > maxSchemaVer {
		if minSchemaVer == maxSchemaVer {
			return fmt.Errorf("database schema version is not compatible with baur version, schema version: %d, expecting version: %d", ver, minSchemaVer)
		}

		return fmt.Errorf("database schema version is not compatible with baur version, schema version: %d, expecting schema version >=%d and <=%d", ver, minSchemaVer, maxSchemaVer)
	}

	return nil
}

func (c *Client) tableExists(ctx context.Context, tableName string) (bool, error) {
	const query = `
	SELECT EXISTS
	       (
		SELECT FROM pg_tables
		 WHERE schemaname = 'public'
		   AND tablename = $1
	       )
`

	var exists bool

	err := c.db.QueryRow(ctx, query, tableName).Scan(&exists)

	return exists, err
}

func (c *Client) v0SchemaNotExits(ctx context.Context) error {
	exists, err := c.tableExists(ctx, "input_build")
	if err != nil {
		return err
	}

	if exists {
		return errors.New("incompatible database schema from baur version <2 found.\n" +
			"Upgrading is unsupported.\n" +
			"Please create a new database.",
		)
	}

	return nil
}

// schemaExist nil if the migrations table exist, otherwise storage.ErrNotExist.
func (c *Client) schemaExist(ctx context.Context) error {
	exists, err := c.tableExists(ctx, "migrations")
	if err != nil {
		return err
	}

	if !exists {
		return storage.ErrNotExist
	}

	return nil
}

func (c *Client) SchemaVersion(ctx context.Context) (int32, error) {
	if err := c.schemaExist(ctx); err != nil {
		return -1, err
	}

	return c.schemaVersion(ctx)
}

func (c *Client) MaxSchemaVersion() int32 {
	return maxSchemaVer
}
