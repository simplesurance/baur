CREATE TABLE input_env_var (
	id serial PRIMARY KEY,
	name text NOT NULL,
	digest text NOT NULL,
	CONSTRAINT input_env_var_name_digest_uniq UNIQUE (digest)
);
CREATE INDEX idx_input_env_var_name ON input_env_var(name);

CREATE TABLE task_run_env_var_input (
	task_run_id integer NOT NULL REFERENCES task_run(id) ON DELETE CASCADE,
	input_env_var_id integer NOT NULL REFERENCES input_env_var(id) ON DELETE CASCADE,
	CONSTRAINT task_run_env_var_input_task_run_id_input_env_var_id_uniq UNIQUE (task_run_id, input_env_var_id)
);
CREATE INDEX idx_task_run_env_var_input ON task_run_env_var_input(task_run_id);
