LOCK TABLE input_file, task_run_file_input IN ACCESS EXCLUSIVE MODE;

ALTER TABLE task_run_file_input
	DROP CONSTRAINT task_run_file_input_input_file_id_fkey;

ALTER TABLE input_file
	ALTER COLUMN id TYPE bigint;

ALTER TABLE task_run_file_input
	ALTER COLUMN input_file_id TYPE bigint;

ALTER SEQUENCE input_file_id_seq AS bigint NO MAXVALUE;

WITH input_file_max_id AS (
	SELECT max(id) AS id FROM input_file
)
SELECT setval('input_file_id_seq', COALESCE(id, 1), id IS NOT NULL)
  FROM input_file_max_id;

ALTER TABLE task_run_file_input
	ADD CONSTRAINT task_run_file_input_input_file_id_fkey
	FOREIGN KEY (input_file_id) REFERENCES input_file(id) ON DELETE CASCADE;
