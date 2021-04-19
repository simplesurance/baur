package postgres

import (
	"context"
	"errors"
	"fmt"
)

const schemaVer = 1

const initQuery = `
CREATE TABLE migrations (
	schema_version integer NOT NULL
);

INSERT INTO migrations (schema_version) VALUES(1);

CREATE TABLE application (
	id serial PRIMARY KEY,
	name text NOT NULL UNIQUE,
	CONSTRAINT application_name_uniq UNIQUE (name)
);

CREATE TABLE vcs (
	id serial PRIMARY KEY,
	revision text NOT NULL,
	dirty boolean NOT NULL,
	CONSTRAINT vcs_revision_dirty_uniq UNIQUE (revision, dirty)
);

CREATE TABLE output (
	id serial PRIMARY KEY,
	name text NOT NULL,
	type text NOT NULL,
	digest text NOT NULL,
	size_bytes bigint NOT NULL CHECK (size_bytes >= 0),
	CONSTRAINT output_name_type_digest_size_bytes_uniq UNIQUE (name, type, digest, size_bytes)
);

CREATE TABLE upload (
	id serial PRIMARY KEY,
	uri text NOT NULL,
	method text NOT NULL,
	start_timestamp timestamp with time zone NOT NULL,
	stop_timestamp timestamp with time zone NOT NULL
);

CREATE TABLE task (
	id serial PRIMARY KEY,
	name text NOT NULL,
	application_id integer NOT NULL REFERENCES application(id) ON DELETE CASCADE,
	CONSTRAINT task_name_application_id_uniq UNIQUE (name, application_id)
);

CREATE TABLE task_run (
	id serial PRIMARY KEY,
	vcs_id integer REFERENCES vcs(id),
	task_id integer NOT NULL REFERENCES task (id) ON DELETE CASCADE,
	start_timestamp timestamp with time zone NOT NULL,
	stop_timestamp timestamp with time zone NOT NULL,
	result text NOT NULL,
	CONSTRAINT result_check CHECK (result in ('success', 'failure'))
);

CREATE TABLE input (
	id serial PRIMARY KEY,
	uri text NOT NULL,
	digest text NOT NULL
);

CREATE UNIQUE INDEX input_uri_digest_uniq ON input (MD5(uri), digest);

CREATE TABLE task_run_input (
	task_run_id integer NOT NULL REFERENCES task_run(id) ON DELETE CASCADE,
	input_id integer NOT NULL REFERENCES input(id) ON DELETE CASCADE,
	total_digest text NOT NULL,
	CONSTRAINT inputs_task_run_id_input_id_uniq UNIQUE(task_run_id, input_id)
);

CREATE INDEX idx_task_run_input_task_run_id ON task_run_input(task_run_id);
CREATE INDEX idx_task_run_input_total_digest ON task_run_input(total_digest);

CREATE TABLE task_run_output (
	task_run_id integer NOT NULL REFERENCES task_run (id) ON DELETE CASCADE,
	output_id integer NOT NULL REFERENCES output (id) ON DELETE CASCADE,
	upload_id integer NOT NULL REFERENCES upload(id) ON DELETE CASCADE,
	CONSTRAINT task_output_task_run_id_output_id_upload_id_uniq UNIQUE (task_run_id, output_id, upload_id)
);

CREATE INDEX idx_task_run_output_task_run_id ON task_run_output(task_run_id);
`

// Init creates the baur tables in the postgresql database
func (c *Client) Init(ctx context.Context) error {
	_, err := c.db.Exec(ctx, initQuery)

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

func (c *Client) ensureSchemaIsCompatible(ctx context.Context) error {
	var rowsCount int

	rows, err := c.db.Query(ctx, "SELECT schema_version from migrations")
	if err != nil {
		return fmt.Errorf("querying schema_version failed: %w", err)
	}

	defer rows.Close()

	for rows.Next() {
		var ver int

		if rowsCount != 0 {
			return errors.New("migrations table contains >1 rows")
		}

		err = rows.Scan(&ver)
		if err != nil {
			return err
		}

		if ver != schemaVer {
			return fmt.Errorf("database schema version is not compatible with baur version, schema version: %d, expected version: %d", ver, schemaVer)
		}

		rowsCount++
	}

	if err := rows.Err(); err != nil {
		return err
	}

	if rowsCount != 1 {
		return fmt.Errorf("read %d rows from migrations table, expected 1", rowsCount)
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
		return errors.New("incompatible database schema from old baur version exists")
	}

	return nil
}

func (c *Client) schemaExist(ctx context.Context) error {
	exists, err := c.tableExists(ctx, "migrations")
	if err != nil {
		return err
	}

	if !exists {
		return errors.New("database schema does not exist")
	}

	return nil
}
