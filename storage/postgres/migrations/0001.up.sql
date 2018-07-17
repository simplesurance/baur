CREATE TABLE application (
	id SERIAL PRIMARY KEY,
	application_name TEXT NOT NULL UNIQUE
);

CREATE TABLE build (
	id SERIAL PRIMARY KEY,
	application_id integer REFERENCES application (id),
	start_timestamp timestamp with time zone,
	stop_timestamp timestamp with time zone,
	total_src_digest TEXT
);

CREATE TABLE artifact (
	id SERIAL PRIMARY KEY,
	name TEXT NOT NULL,
	type TEXT,
	digest TEXT,
	size_bytes integer,
	CONSTRAINT artifact_uniq UNIQUE(name, digest, size_bytes)
);

CREATE TABLE artifact_build (
	build_id integer REFERENCES build (id) NOT NULL,
	artifact_id integer REFERENCES artifact (id) NOT NULL
	CONSTRAINT artifact_build_uniq UNIQUE(build_id, artifact_id)
);

CREATE TABLE upload (
	id SERIAL PRIMARY KEY,
	artifact_id integer REFERENCES artifact (id) NOT NULL,
	uri TEXT, /* TODO: should this be unique? */
	upload_duration_msec integer
);

CREATE TABLE source (
	id SERIAL PRIMARY KEY,
	relative_path TEXT NOT NULL,
	digest TEXT NOT NULL,
	CONSTRAINT source_uniq UNIQUE(relative_path, digest)
);

CREATE TABLE source_build (
	build_id integer REFERENCES build (id),
	source_id integer REFERENCES source(id)
	CONSTRAINT source_build_uniq UNIQUE(build_id, source_id)
);
