CREATE TABLE application (
	id SERIAL PRIMARY KEY,
	application_name TEXT NOT NULL UNIQUE
);

CREATE TABLE build (
	id SERIAL PRIMARY KEY,
	application_id integer REFERENCES application (id),
	start_timestamp timestamp with time zone,
	stop_timestamp timestamp with time zone,
	total_src_hash TEXT
);

CREATE TABLE artifact (
	id SERIAL PRIMARY KEY,
	build_id integer REFERENCES build (id) NOT NULL,
	name TEXT NOT NULL,
	type TEXT,
	url TEXT,
	hash TEXT,
	size_bytes integer,
	upload_duration_msec integer
);

CREATE TABLE source (
	id SERIAL PRIMARY KEY,
	relative_path TEXT NOT NULL,
	hash TEXT NOT NULL,
	CONSTRAINT source_uniq UNIQUE(relative_path, hash)
);

CREATE TABLE source_build (
	build_id integer REFERENCES build (id),
	source_id integer REFERENCES source(id)
);
