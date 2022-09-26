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
	total_input_digest text NOT NULL,
	start_timestamp timestamp with time zone NOT NULL,
	stop_timestamp timestamp with time zone NOT NULL,
	result text NOT NULL,
	CONSTRAINT result_check CHECK (result in ('success', 'failure'))
);
CREATE INDEX idx_task_run_total_input_digest ON task_run(total_input_digest);

CREATE TABLE input_file (
	id serial PRIMARY KEY,
	path text NOT NULL,
	digest text NOT NULL,
	CONSTRAINT input_file_path_digest_uniq UNIQUE (path, digest)
);
CREATE INDEX idx_input_file_path ON input_file(path);

CREATE TABLE input_string (
	id serial PRIMARY KEY,
	string text NOT NULL,
	digest text NOT NULL,
	CONSTRAINT input_string_digest_uniq UNIQUE (digest)
);
/* An index on the input_string.string column would limit the size of the
  values to the max. size of columns in indexes (8191B).
*/

CREATE TABLE task_run_file_input (
	task_run_id integer NOT NULL REFERENCES task_run(id) ON DELETE CASCADE,
	input_file_id integer NOT NULL REFERENCES input_file(id) ON DELETE CASCADE,
	CONSTRAINT task_run_file_input_task_run_id_input_id_uniq UNIQUE (task_run_id, input_file_id)
);
CREATE INDEX task_run_file_input_task_run_id_idx ON task_run_file_input(task_run_id);

CREATE TABLE task_run_string_input (
	task_run_id integer NOT NULL REFERENCES task_run(id) ON DELETE CASCADE,
	input_string_id integer NOT NULL REFERENCES input_string(id) ON DELETE CASCADE,
	CONSTRAINT task_run_string_input_task_run_id_input_string_id_uniq UNIQUE (task_run_id, input_string_id)
);
CREATE INDEX idx_task_run_string_input ON task_run_string_input(task_run_id);

CREATE TABLE task_run_output (
	task_run_id integer NOT NULL REFERENCES task_run (id) ON DELETE CASCADE,
	output_id integer NOT NULL REFERENCES output (id) ON DELETE CASCADE,
	upload_id integer NOT NULL REFERENCES upload(id) ON DELETE CASCADE,
	CONSTRAINT task_output_task_run_id_output_id_upload_id_uniq UNIQUE (task_run_id, output_id, upload_id)
);

CREATE INDEX idx_task_run_output_task_run_id ON task_run_output(task_run_id);
