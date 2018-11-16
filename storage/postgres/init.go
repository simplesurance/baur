package postgres

const initQuery = `
CREATE TABLE application (
	id SERIAL PRIMARY KEY,
	name TEXT NOT NULL UNIQUE
);

CREATE TABLE vcs (
	id SERIAL PRIMARY KEY,
	commit TEXT NOT NULL,
	dirty BOOL NOT NULL,
	CONSTRAINT vcs_uniq UNIQUE(commit, dirty)
);

CREATE TABLE build (
	id SERIAL PRIMARY KEY,
	vcs_id INTEGER REFERENCES vcs(id) ON DELETE CASCADE,
	application_id INTEGER REFERENCES application (id) ON DELETE CASCADE,
	start_timestamp TIMESTAMP WITH TIME ZONE NOT NULL,
	stop_timestamp TIMESTAMP WITH TIME ZONE NOT NULL,
	total_input_digest TEXT NOT NULL
);

CREATE TABLE output (
	id SERIAL PRIMARY KEY,
	name TEXT NOT NULL,
	type TEXT NOT NULL,
	digest TEXT NOT NULL UNIQUE,
	size_bytes INTEGER NOT NULL
);

CREATE TABLE build_output (
	id SERIAL PRIMARY KEY,
	build_id INTEGER REFERENCES build (id) ON DELETE CASCADE,
	output_id INTEGER REFERENCES output (id) ON DELETE CASCADE,
	CONSTRAINT build_output_uniq UNIQUE(build_id, output_id)
);

CREATE TABLE upload (
	id SERIAL PRIMARY KEY,
	build_output_id INTEGER REFERENCES build_output (id) ON DELETE CASCADE,
	uri TEXT NOT NULL,
	method TEXT NOT NULL,
	upload_duration_ns BIGINT NOT NULL
);

CREATE TABLE input (
	id SERIAL PRIMARY KEY,
	uri TEXT NOT NULL,
	digest TEXT NOT NULL,
	CONSTRAINT input_uniq UNIQUE(uri, digest)
);

CREATE TABLE input_build (
	build_id INTEGER REFERENCES build (id) ON DELETE CASCADE,
	input_id INTEGER REFERENCES input(id) ON DELETE CASCADE,
	CONSTRAINT input_build_uniq UNIQUE(build_id, input_id)
);
`

// Init creates the baur tables in the postgresql database
func (c *Client) Init() error {
	_, err := c.Db.Exec(initQuery)

	return err
}
