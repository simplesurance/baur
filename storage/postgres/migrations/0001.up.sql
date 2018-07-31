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
	vcs_id INTEGER REFERENCES vcs(id),
	application_id integer REFERENCES application (id),
	start_timestamp timestamp with time zone,
	stop_timestamp timestamp with time zone,
	total_input_digest TEXT
);

CREATE TABLE output (
	id SERIAL PRIMARY KEY,
	name TEXT NOT NULL,
	type TEXT,
	digest TEXT UNIQUE,
	size_bytes integer
);

CREATE TABLE build_output (
	id SERIAL PRIMARY KEY,
	build_id INTEGER REFERENCES build (id) NOT NULL,
	output_id INTEGER REFERENCES output (id) NOT NULL,
	CONSTRAINT build_output_uniq UNIQUE(build_id, output_id)
);

CREATE TABLE upload (
	id SERIAL PRIMARY KEY,
	build_output_id integer REFERENCES build_output (id) NOT NULL,
	uri TEXT, /* TODO: should this be unique? */
	upload_duration_msec integer
);

CREATE TABLE input (
	id SERIAL PRIMARY KEY,
	url TEXT NOT NULL,
	digest TEXT NOT NULL,
	CONSTRAINT input_uniq UNIQUE(url, digest)
);

CREATE TABLE input_build (
	build_id integer REFERENCES build (id),
	input_id integer REFERENCES input(id),
	CONSTRAINT input_build_uniq UNIQUE(build_id, input_id)
);
