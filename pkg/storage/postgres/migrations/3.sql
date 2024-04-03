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
