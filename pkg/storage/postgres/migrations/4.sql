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
