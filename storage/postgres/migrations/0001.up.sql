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
	application_id INTEGER REFERENCES application (id),
	start_timestamp TIMESTAMP WITH TIME ZONE,
	stop_timestamp TIMESTAMP WITH TIME ZONE,
	total_input_digest TEXT
);

CREATE TABLE output (
	id SERIAL PRIMARY KEY,
	name TEXT NOT NULL,
	type TEXT NOT NULL,
	digest TEXT UNIQUE,
	size_bytes INTEGER
);

CREATE TABLE build_output (
	id SERIAL PRIMARY KEY,
	build_id INTEGER REFERENCES build (id) NOT NULL,
	output_id INTEGER REFERENCES output (id) NOT NULL,
	CONSTRAINT build_output_uniq UNIQUE(build_id, output_id)
);

CREATE TABLE upload (
	id SERIAL PRIMARY KEY,
	build_output_id INTEGER REFERENCES build_output (id) NOT NULL,
	url TEXT NOT NULL,
	upload_duration_ns BIGINT NOT NULL
);

CREATE TABLE input (
	id SERIAL PRIMARY KEY,
	url TEXT NOT NULL,
	digest TEXT NOT NULL,
	CONSTRAINT input_uniq UNIQUE(url, digest)
);

CREATE TABLE input_build (
	build_id INTEGER REFERENCES build (id),
	input_id INTEGER REFERENCES input(id),
	CONSTRAINT input_build_uniq UNIQUE(build_id, input_id)
);
