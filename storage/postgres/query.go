package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v4"

	"github.com/simplesurance/baur/v1/storage"
)

func (c *Client) TaskRun(ctx context.Context, id int) (*storage.TaskRunWithID, error) {
	var taskRun *storage.TaskRunWithID

	idFilter := []*storage.Filter{
		{
			Field:    storage.FieldID,
			Operator: storage.OpEQ,
			Value:    id,
		},
	}

	err := c.TaskRuns(ctx, idFilter, nil, func(tr *storage.TaskRunWithID) error {
		taskRun = tr

		return nil
	})

	if err != nil {
		return nil, err
	}

	if taskRun == nil {
		panic("TaskRuns returned a nil TaskRunWithID and nil error")
	}

	return taskRun, nil
}

func (c *Client) LatestTaskRunByDigest(ctx context.Context, appName, taskName, totalInputDigest string) (*storage.TaskRunWithID, error) {
	// TODO: improve the query, retrieving the newest record via LIMIT should be slow
	const query = `
	SELECT task_run.id,
	       application.name,
	       task.name,
	       vcs.revision,
	       vcs.dirty,
	       task_run_input.total_digest,
	       task_run.start_timestamp,
	       task_run.stop_timestamp,
	       task_run.result
	  FROM application
	  JOIN task ON application.id = task.application_id
	  JOIN task_run ON task.id = task_run.task_id
	  JOIN task_run_input ON task_run_input.task_run_id = task_run.id
	  LEFT OUTER JOIN vcs ON vcs.id = task_run.vcs_id
	 WHERE application.name = $1
	   AND task.name = $2
	   AND task_run_input.total_digest = $3
	 ORDER BY task_run.stop_timestamp DESC
	 LIMIT 1
	 `

	var result storage.TaskRunWithID

	row := c.db.QueryRow(ctx, query, appName, taskName, totalInputDigest)

	err := row.Scan(
		&result.ID,
		&result.ApplicationName,
		&result.TaskName,
		&result.VCSRevision,
		&result.VCSIsDirty,
		&result.TotalInputDigest,
		&result.StartTimestamp,
		&result.StopTimestamp,
		&result.Result,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, storage.ErrNotExist
		}

		return nil, fmt.Errorf("query %s with args: %s failed: %w", query, strArgList(appName, taskName, totalInputDigest), err)
	}

	return &result, nil
}

func (c *Client) Inputs(ctx context.Context, taskRunID int) ([]*storage.Input, error) {
	const query = `
	SELECT input.uri,
	       input.digest
	  FROM input
	  JOIN task_run_input ON input.id = task_run_input.input_id
	  WHERE task_run_input.task_run_id = $1
	  `

	var result []*storage.Input

	rows, err := c.db.Query(ctx, query, taskRunID)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, storage.ErrNotExist
		}

		return nil, fmt.Errorf("query %s with arg: %d failed: %w", query, taskRunID, err)
	}

	for rows.Next() {
		var input storage.Input

		if err := rows.Scan(&input.URI, &input.Digest); err != nil {
			rows.Close()
			return nil, fmt.Errorf("query %s with arg: %d failed: %w", query, taskRunID, err)
		}

		result = append(result, &input)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("query %s with arg: %d failed: %w", query, taskRunID, err)
	}

	if len(result) == 0 {
		return nil, storage.ErrNotExist
	}

	return result, nil
}

func (c *Client) Outputs(ctx context.Context, taskRunID int) ([]*storage.Output, error) {
	const query = `
	SELECT output.id,
	       output.name,
	       output.type,
	       output.digest,
	       output.size_bytes,
	       upload.uri,
	       upload.method,
	       upload.start_timestamp,
	       upload.stop_timestamp
	  FROM output
	  JOIN task_run_output ON task_run_output.output_id = output.id
	  JOIN upload ON upload.id = task_run_output.upload_id
	 WHERE task_run_output.task_run_id = $1
	 `

	resMap := map[int]*storage.Output{}

	rows, err := c.db.Query(ctx, query, taskRunID)
	if err != nil {
		return nil, fmt.Errorf("query %s with arg: %d failed: %w", query, taskRunID, err)
	}

	for rows.Next() {
		var upload storage.Upload
		var outputID int
		output := &storage.Output{}

		err := rows.Scan(
			&outputID,
			&output.Name,
			&output.Type,
			&output.Digest,
			&output.SizeBytes,
			&upload.URI,
			&upload.Method,
			&upload.UploadStartTimestamp,
			&upload.UploadStopTimestamp,
		)
		if err != nil {
			rows.Close()
			return nil, fmt.Errorf("query %s with arg: %d failed: %w", query, taskRunID, err)
		}

		if rec := resMap[outputID]; rec == nil {
			resMap[outputID] = output
		} else {
			output = rec
		}

		output.Uploads = append(output.Uploads, &upload)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("query %s with arg: %d failed: %w", query, taskRunID, err)
	}

	if len(resMap) == 0 {
		return nil, storage.ErrNotExist
	}

	result := make([]*storage.Output, 0, len(resMap))
	for _, output := range resMap {
		result = append(result, output)
	}

	return result, nil
}

func (c *Client) TaskRuns(
	ctx context.Context,
	filters []*storage.Filter,
	sorters []*storage.Sorter,
	cb func(*storage.TaskRunWithID) error,
) error {
	const queryStr = `
	SELECT task_run_id, application_name, task_name, revision, dirty, total_digest, start_timestamp, stop_timestamp, result
	  FROM (
	       SELECT DISTINCT ON (task_run.id)
		      task_run.id AS task_run_id,
	              application.name AS application_name,
	              task.name AS task_name,
	              vcs.revision,
	              vcs.dirty,
	              task_run_input.total_digest,
	              task_run.start_timestamp AS start_timestamp,
	              task_run.stop_timestamp,
	              task_run.result,
	              (EXTRACT(EPOCH FROM (task_run.stop_timestamp - task_run.start_timestamp))::bigint * 1000000000) AS duration
	         FROM application
	         JOIN task ON application.id = task.application_id
	         JOIN task_run ON task.id = task_run.task_id
	         JOIN task_run_input ON task_run_input.task_run_id = task_run.id
	         LEFT OUTER JOIN vcs ON vcs.id = task_run.vcs_id
	       ) tr
	  `

	return taskRuns(ctx, queryStr, c.db, filters, sorters, cb)
}

func (c *Client) TaskRunsWithInputURI(
	ctx context.Context,
	filters []*storage.Filter,
	sorters []*storage.Sorter,
	digest string,
	cb func(*storage.TaskRunWithID) error,
) error {
	const queryStr = `
	SELECT task_run_id, application_name, task_name, revision, dirty, total_digest, start_timestamp, stop_timestamp, result
	  FROM (
	       SELECT DISTINCT ON (task_run.id, input.uri)
		      task_run.id AS task_run_id,
		      application.name AS application_name,
		      task.name AS task_name,
		      vcs.revision,
		      vcs.dirty,
		      task_run_input.total_digest,
		      task_run.start_timestamp AS start_timestamp,
		      task_run.stop_timestamp,
		      task_run.result,
		      input.uri AS uri,
		      (EXTRACT(EPOCH FROM (task_run.stop_timestamp - task_run.start_timestamp))::bigint * 1000000000) AS duration
		 FROM application
		 JOIN task ON application.id = task.application_id
		 JOIN task_run ON task.id = task_run.task_id
		 JOIN task_run_input ON task_run_input.task_run_id = task_run.id
		 JOIN input ON task_run_input.input_id = input.id
		 LEFT OUTER JOIN vcs ON vcs.id = task_run.vcs_id
	       ) tr
	  `

	filters = append(filters, &storage.Filter{
		Field:    storage.FieldURI,
		Operator: storage.OpEQ,
		Value:    digest,
	})

	return taskRuns(ctx, queryStr, c.db, filters, sorters, cb)
}

func taskRuns(
	ctx context.Context,
	queryStr string,
	db dbConn,
	filters []*storage.Filter,
	sorters []*storage.Sorter,
	cb func(*storage.TaskRunWithID) error,
) error {
	var queryReturnedRows bool

	q := query{
		BaseQuery: queryStr,
		Filters:   filters,
		Sorters:   sorters,
	}

	query, args, err := q.Compile()
	if err != nil {
		return fmt.Errorf("compiling query string failed: %w", err)
	}

	rows, err := db.Query(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("query %s with args: %s failed: %w", query, strArgList(args), err)
	}

	for rows.Next() {
		var taskRun storage.TaskRunWithID

		queryReturnedRows = true

		err := rows.Scan(
			&taskRun.ID,
			&taskRun.ApplicationName,
			&taskRun.TaskName,
			&taskRun.VCSRevision,
			&taskRun.VCSIsDirty,
			&taskRun.TotalInputDigest,
			&taskRun.StartTimestamp,
			&taskRun.StopTimestamp,
			&taskRun.Result,
		)

		if err != nil {
			rows.Close()
			return fmt.Errorf("query %s with args: %s failed: %w", query, strArgList(args), err)
		}

		if err := cb(&taskRun); err != nil {
			rows.Close()
			return fmt.Errorf("callback failed: %w", err)
		}
	}

	if err := rows.Err(); err != nil {
		return fmt.Errorf("query %s with args: %s failed: %w", query, strArgList(args), err)
	}

	if !queryReturnedRows {
		return storage.ErrNotExist
	}

	return nil
}
