CREATE TABLE input_task (
	id serial PRIMARY KEY,
	name text NOT NULL,
	digest text NOT NULL,
	CONSTRAINT input_task_name_digest_uniq UNIQUE (name, digest)
);

CREATE TABLE task_run_task_input (
	task_run_id integer NOT NULL REFERENCES task_run(id) ON DELETE CASCADE,
	input_task_id integer NOT NULL REFERENCES input_task(id) ON DELETE CASCADE,
	CONSTRAINT task_run_task_input_task_run_id_input_task_id_uniq UNIQUE (task_run_id, input_task_id)
);

CREATE TABLE release (
	id serial PRIMARY KEY,
	name text NOT NULL,
	metadata bytea,
	CONSTRAINT release_name_uniq UNIQUE (name)
);

CREATE TABLE release_task_run (
	release_id integer NOT NULL REFERENCES release (id) ON DELETE CASCADE,
	task_run_id integer NOT NULL REFERENCES task_run (id) ON DELETE CASCADE,
	PRIMARY KEY(release_id, task_run_id)
);

DROP INDEX task_run_file_input_task_run_id_idx;
DROP INDEX idx_task_run_string_input;
DROP INDEX idx_task_run_output_task_run_id;
