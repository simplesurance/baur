package postgres

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v4"

	"github.com/simplesurance/baur/v5/pkg/storage"
)

func (c *Client) ReleasesDelete(ctx context.Context, before time.Time, pretend bool) (*storage.ReleasesDeleteResult, error) {
	var result storage.ReleasesDeleteResult

	err := c.db.BeginFunc(ctx, func(tx pgx.Tx) error {
		var err error
		if pretend {
			defer tx.Rollback(ctx) //nolint: errcheck
		}

		result.DeletedReleases, err = c.deleteOldReleases(ctx, tx, before)
		if err != nil {
			return err
		}

		return nil
	})

	if err != nil && !(pretend && errors.Is(err, pgx.ErrTxClosed)) {
		return nil, err
	}

	return &result, nil
}

func (*Client) deleteOldReleases(ctx context.Context, tx pgx.Tx, before time.Time) (int64, error) {
	const query = `
	      DELETE FROM release
	       WHERE created_at < $1
	`

	t, err := tx.Exec(ctx, query, before)
	if err != nil {
		return 0, newQueryError(query, err, before)
	}

	return t.RowsAffected(), nil
}

func (c *Client) TaskRunsDelete(ctx context.Context, before time.Time, pretend bool) (*storage.TaskRunsDeleteResult, error) {
	var result storage.TaskRunsDeleteResult

	err := c.db.BeginFunc(ctx, func(tx pgx.Tx) error {
		var err error
		if pretend {
			defer tx.Rollback(ctx) //nolint: errcheck
		}

		result.DeletedTaskRuns, err = c.deleteOldTaskRuns(ctx, tx, before)
		if err != nil {
			return err
		}

		result.DeletedTasks, err = c.deleteUnusedTasks(ctx, tx)
		if err != nil {
			return err
		}

		result.DeletedApps, err = c.deleteUnusedApps(ctx, tx)
		if err != nil {
			return err
		}

		result.DeletedOutputs, err = c.deleteUnusedOutputs(ctx, tx)
		if err != nil {
			return err
		}

		result.DeletedUploads, err = c.deleteUnusedUploads(ctx, tx)
		if err != nil {
			return err
		}

		result.DeletedInputs, err = c.deleteUnusedInputs(ctx, tx)
		if err != nil {
			return err
		}

		result.DeletedVCS, err = c.deleteUnusedVCS(ctx, tx)
		if err != nil {
			return err
		}

		return nil
	})

	if err != nil && !(pretend && errors.Is(err, pgx.ErrTxClosed)) {
		return nil, err
	}

	return &result, nil
}

func (*Client) deleteOldTaskRuns(ctx context.Context, tx pgx.Tx, before time.Time) (int64, error) {
	const query = `
	      DELETE FROM task_run
	       WHERE start_timestamp < $1
	         AND NOT EXISTS (
			SELECT 1 FROM release_task_run
			 WHERE task_run.id = release_task_run.task_run_id
	       )
	`

	t, err := tx.Exec(ctx, query, before)
	if err != nil {
		return 0, newQueryError(query, err, before)
	}

	return t.RowsAffected(), nil
}

func (*Client) deleteUnusedTasks(ctx context.Context, tx pgx.Tx) (int64, error) {
	const query = `
		DELETE FROM task
	 	 WHERE NOT EXISTS (
			SELECT 1 FROM task_run
			 WHERE task.id = task_run.task_id
		 )
		`
	t, err := tx.Exec(ctx, query)
	if err != nil {
		return 0, newQueryError(query, err)
	}

	return t.RowsAffected(), nil
}

func (*Client) deleteUnusedApps(ctx context.Context, tx pgx.Tx) (int64, error) {
	const query = `
		DELETE FROM application
	 	 WHERE id NOT IN (
			 SELECT task.application_id FROM task
		 )
		`
	t, err := tx.Exec(ctx, query)
	if err != nil {
		return 0, newQueryError(query, err)
	}

	return t.RowsAffected(), nil
}

func (*Client) deleteUnusedOutputs(ctx context.Context, tx pgx.Tx) (int64, error) {
	const query = `
		DELETE FROM output
	 	 WHERE id NOT IN (
			 SELECT task_run_output.output_id
			   FROM task_run_output
		 )
		`
	t, err := tx.Exec(ctx, query)
	if err != nil {
		return 0, newQueryError(query, err)
	}

	return t.RowsAffected(), nil
}

func (*Client) deleteUnusedUploads(ctx context.Context, tx pgx.Tx) (int64, error) {
	const query = `
		DELETE FROM upload
	 	 WHERE id NOT IN (
			 SELECT task_run_output.upload_id
			   FROM task_run_output
		 )
		`
	t, err := tx.Exec(ctx, query)
	if err != nil {
		return 0, newQueryError(query, err)
	}

	return t.RowsAffected(), nil
}

func (*Client) deleteUnusedVCS(ctx context.Context, tx pgx.Tx) (int64, error) {
	const query = `
		DELETE FROM vcs
	 	 WHERE NOT EXISTS (
			 SELECT 1 FROM task_run
			  WHERE vcs.id = task_run.vcs_id
		 )
		`
	t, err := tx.Exec(ctx, query)
	if err != nil {
		return 0, newQueryError(query, err)
	}

	return t.RowsAffected(), nil
}

func (*Client) deleteUnusedInputs(ctx context.Context, tx pgx.Tx) (int64, error) {
	var cnt int64
	const qInputFiles = `
		DELETE FROM input_file
	 	 WHERE NOT EXISTS (
			SELECT 1 FROM task_run_file_input
			 WHERE input_file.id = task_run_file_input.input_file_id
		 )
		`
	t, err := tx.Exec(ctx, qInputFiles)
	if err != nil {
		return 0, newQueryError(qInputFiles, err)
	}
	cnt = t.RowsAffected()

	const qInputStrings = `
		DELETE FROM input_string
	 	 WHERE id NOT IN (
			 SELECT task_run_string_input.input_string_id
			   FROM task_run_string_input
		 )
		`
	t, err = tx.Exec(ctx, qInputStrings)
	if err != nil {
		return 0, newQueryError(qInputFiles, err)
	}
	cnt += t.RowsAffected()

	const qInputTasks = `
		DELETE FROM input_task
	 	 WHERE id NOT IN (
			 SELECT task_run_task_input.input_task_id
			   FROM task_run_task_input
		 )
		`
	t, err = tx.Exec(ctx, qInputTasks)
	if err != nil {
		return 0, newQueryError(qInputFiles, err)
	}
	cnt += t.RowsAffected()

	return cnt, nil
}
