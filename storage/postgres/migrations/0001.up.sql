CREATE TABLE application (
	id SERIAL PRIMARY KEY,
	name TEXT NOT NULL UNIQUE
);

CREATE TABLE build (
	id SERIAL PRIMARY KEY,
	application_id integer REFERENCES application (id),
	start_timestamp timestamp with time zone,
	stop_timestamp timestamp with time zone,
	total_input_digest TEXT
);

CREATE TABLE output (
	id SERIAL PRIMARY KEY,
	name TEXT NOT NULL,
	type TEXT,
	digest TEXT,
	size_bytes integer,
	CONSTRAINT output_uniq UNIQUE(name, digest, size_bytes)
);

CREATE TABLE upload (
	id SERIAL PRIMARY KEY,
	build_id integer REFERENCES build (id) NOT NULL,
	output_id integer REFERENCES output (id) NOT NULL,
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
