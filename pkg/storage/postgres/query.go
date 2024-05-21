package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v4"

	"github.com/simplesurance/baur/v3/pkg/storage"
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

	err := c.TaskRuns(ctx, idFilter, nil, storage.NoLimit, func(tr *storage.TaskRunWithID) error {
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
	       task_run.total_input_digest,
	       task_run.start_timestamp,
	       task_run.stop_timestamp,
	       task_run.result
	  FROM application
	  JOIN task ON application.id = task.application_id
	  JOIN task_run ON task.id = task_run.task_id
	  LEFT OUTER JOIN vcs ON vcs.id = task_run.vcs_id
	 WHERE application.name = $1
	   AND task.name = $2
	   AND task_run.total_input_digest = $3
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

func (c *Client) inputStrings(ctx context.Context, taskRunID int) ([]*storage.InputString, error) {
	const query = `
	SELECT input_string.string,
	       input_string.digest
	  FROM input_string
	  JOIN task_run_string_input ON input_string.id = task_run_string_input.input_string_id
	 WHERE task_run_string_input.task_run_id = $1
	 `

	var result []*storage.InputString

	rows, err := c.db.Query(ctx, query, taskRunID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, storage.ErrNotExist
		}

		return nil, fmt.Errorf("query %s with arg: %d failed: %w", query, taskRunID, err)
	}

	for rows.Next() {
		var input storage.InputString

		if err := rows.Scan(&input.String, &input.Digest); err != nil {
			rows.Close()
			return nil, fmt.Errorf("query %s with arg: %d failed: %w", query, taskRunID, err)
		}

		result = append(result, &input)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("query %s with arg: %d failed: %w", query, taskRunID, err)
	}

	return result, nil

}

func (c *Client) inputFiles(ctx context.Context, taskRunID int) ([]*storage.InputFile, error) {
	const query = `
	SELECT input_file.path,
	       input_file.digest
	  FROM input_file
	  JOIN task_run_file_input ON input_file.id = task_run_file_input.input_file_id
         WHERE task_run_file_input.task_run_id = $1
	  `

	var result []*storage.InputFile

	rows, err := c.db.Query(ctx, query, taskRunID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, storage.ErrNotExist
		}

		return nil, fmt.Errorf("query %s with arg: %d failed: %w", query, taskRunID, err)
	}

	for rows.Next() {
		var input storage.InputFile

		if err := rows.Scan(&input.Path, &input.Digest); err != nil {
			rows.Close()
			return nil, fmt.Errorf("query %s with arg: %d failed: %w", query, taskRunID, err)
		}

		result = append(result, &input)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("query %s with arg: %d failed: %w", query, taskRunID, err)
	}

	return result, nil
}

func (c *Client) inputEnvVars(ctx context.Context, taskRunID int) ([]*storage.InputEnvVar, error) {
	const query = `
	SELECT input_env_var.name,
	       input_env_var.digest
	  FROM input_env_var
	  JOIN task_run_env_var_input ON input_env_var.id = task_run_env_var_input.input_env_var_id
         WHERE task_run_env_var_input.task_run_id = $1
	  `

	var result []*storage.InputEnvVar

	rows, err := c.db.Query(ctx, query, taskRunID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, storage.ErrNotExist
		}

		return nil, fmt.Errorf("query %s with arg: %d failed: %w", query, taskRunID, err)
	}

	for rows.Next() {
		var input storage.InputEnvVar

		if err := rows.Scan(&input.Name, &input.Digest); err != nil {
			rows.Close()
			return nil, fmt.Errorf("query %s with arg: %d failed: %w", query, taskRunID, err)
		}

		result = append(result, &input)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("query %s with arg: %d failed: %w", query, taskRunID, err)
	}

	return result, nil
}

func (c *Client) inputTasks(ctx context.Context, taskRunID int) ([]*storage.InputTaskInfo, error) {
	const query = `
	SELECT input_task.name,
	       input_task.digest
	 FROM  input_task
	 JOIN  task_run_task_input ON input_task.id = task_run_task_input.input_task_id
        WHERE  task_run_task_input.task_run_id = $1`

	var result []*storage.InputTaskInfo

	rows, err := c.db.Query(ctx, query, taskRunID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, storage.ErrNotExist
		}

		return nil, fmt.Errorf("query %s with arg: %d failed: %w", query, taskRunID, err)
	}

	for rows.Next() {
		var input storage.InputTaskInfo

		if err := rows.Scan(&input.Name, &input.Digest); err != nil {
			rows.Close()
			return nil, fmt.Errorf("query %s with arg: %d failed: %w", query, taskRunID, err)
		}

		result = append(result, &input)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("query %s with arg: %d failed: %w", query, taskRunID, err)
	}

	return result, nil
}

func (c *Client) Inputs(ctx context.Context, taskRunID int) (*storage.Inputs, error) {
	var result storage.Inputs
	var err error

	result.Files, err = c.inputFiles(ctx, taskRunID)
	if err != nil {
		return nil, err
	}

	result.Strings, err = c.inputStrings(ctx, taskRunID)
	if err != nil {
		return nil, err
	}

	result.EnvironmentVariables, err = c.inputEnvVars(ctx, taskRunID)
	if err != nil {
		return nil, err
	}

	result.TaskInfo, err = c.inputTasks(ctx, taskRunID)
	if err != nil {
		return nil, err
	}

	if len(result.Files) == 0 && len(result.Strings) == 0 && len(result.EnvironmentVariables) == 0 {
		return nil, storage.ErrNotExist
	}

	return &result, nil
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
	limit uint,
	cb func(*storage.TaskRunWithID) error,
) error {
	const queryTemplate = `
	SELECT task_run_id, application_name, task_name, revision, dirty, total_input_digest, start_timestamp, stop_timestamp, result
	  FROM (
	       SELECT DISTINCT ON ({distinct_on})
		      task_run.id AS task_run_id,
	              application.name AS application_name,
	              task.name AS task_name,
	              vcs.revision,
	              vcs.dirty,
	              task_run.total_input_digest,
	              task_run.start_timestamp AS start_timestamp,
	              task_run.stop_timestamp,
	              task_run.result,
	              {fields}
	              (EXTRACT(EPOCH FROM (task_run.stop_timestamp - task_run.start_timestamp))::bigint * 1000000000) AS duration
	         FROM application
	         JOIN task ON application.id = task.application_id
	         JOIN task_run ON task.id = task_run.task_id
		 {joins}
	         LEFT OUTER JOIN vcs ON vcs.id = task_run.vcs_id
	       ) tr
	  `

	containsInputStringFilter := false
	containsInputFileFilter := false
	for _, filter := range filters {
		if filter.Field == storage.FieldInputString {
			containsInputStringFilter = true
		} else if filter.Field == storage.FieldInputFilePath {
			containsInputFileFilter = true
		}
	}

	if containsInputFileFilter && containsInputStringFilter {
		return errors.New("either a FieldInputString or FieldInputFilePath filter can be specified, not both")
	}

	var replacer *strings.Replacer
	if containsInputStringFilter { //nolint: gocritic // ifElseChain: rewrite if-else to switch statement
		replacer = strings.NewReplacer(
			"{distinct_on}", "task_run.id, input_string.string",
			"{fields}", "input_string.string AS input_string_val,",
			"{joins}", "JOIN task_run_string_input ON task_run_string_input.task_run_id = task_run.id\n"+
				"JOIN input_string ON input_string.id = task_run_string_input.input_string_id",
		)
	} else if containsInputFileFilter {
		replacer = strings.NewReplacer(
			"{distinct_on}", "task_run.id, input_file.path",
			"{fields}", "input_file.path AS input_file_path,",
			"{joins}", "JOIN task_run_file_input ON task_run_file_input.task_run_id = task_run.id\n"+
				"JOIN input_file ON input_file.id = task_run_file_input.input_file_id",
		)
	} else {
		replacer = strings.NewReplacer(
			"{distinct_on}", "task_run.id",
			"{fields}", "",
			"{joins}", "")
	}
	queryStr := replacer.Replace(queryTemplate)

	var queryReturnedRows bool

	q := query{
		BaseQuery: queryStr,
		Filters:   filters,
		Sorters:   sorters,
		Limit:     limit,
	}

	query, args, err := q.Compile()
	if err != nil {
		return fmt.Errorf("compiling query string failed: %w", err)
	}

	rows, err := c.db.Query(ctx, query, args...)
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

func (c *Client) ReleaseExists(ctx context.Context, name string) (bool, error) {
	const query = `
	SELECT COUNT(id)
	  FROM release
	 WHERE name = $1
	 LIMIT 1
	 `
	var count int
	err := c.db.QueryRow(ctx, query, name).Scan(&count)
	if err != nil {
		return false, newQueryError(query, err, name)
	}

	return count == 1, nil
}

func (c *Client) ReleaseTaskRuns(ctx context.Context, releaseName string) ([]*storage.ReleaseTaskRunsResult, error) {
	const query = `
		WITH fr AS (
		    SELECT id
		    FROM release
		    WHERE name = $1
		)
		SELECT application.name,
		       task.name,
		       task_run.id,
		       output.id, output.name,
		       upload.uri, upload.method
		FROM fr
		JOIN release_task_run ON release_task_run.release_id = fr.id
		JOIN task_run ON task_run.id = release_task_run.task_run_id
		JOIN task ON task.id = task_run.task_id
		JOIN application ON application.id = task.application_id
	   LEFT	JOIN task_run_output ON task_run_output.task_run_id = release_task_run.task_run_id
	   LEFT JOIN output ON output.id = task_run_output.output_id
	   LEFT JOIN upload ON upload.id = task_run_output.upload_id
	    ORDER BY application.name, task.name, output.name, upload.uri
		`
	rows, err := c.db.Query(ctx, query, releaseName)
	if err != nil {
		return nil, newQueryError(query, err, releaseName)
	}

	var result []*storage.ReleaseTaskRunsResult

	for rows.Next() {
		var r storage.ReleaseTaskRunsResult
		var outputID sql.NullInt32
		var outputName sql.NullString
		var uri sql.NullString
		var uploadMethod sql.NullString

		err := rows.Scan(
			&r.AppName,
			&r.TaskName,
			&r.RunID,
			&outputID,
			&outputName,
			&uri,
			&uploadMethod,
		)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return nil, storage.ErrNotExist
			}
			return nil, newQueryError(query, err, releaseName)
		}

		r.OutputID = int(outputID.Int32)
		r.OutputName = outputName.String
		r.URI = uri.String
		r.UploadMethod = storage.UploadMethod(uploadMethod.String)
		result = append(result, &r)
	}

	if rows.Err() != nil {
		return nil, newQueryError(query, err, releaseName)
	}

	if len(result) == 0 {
		return nil, storage.ErrNotExist
	}

	return result, nil
}

func (c *Client) ReleaseMetadata(ctx context.Context, releaseName string) ([]byte, error) {
	const query = `
	SELECT metadata
	  FROM release
	 WHERE release.name = $1
	 `
	var metadata []byte

	err := c.db.QueryRow(ctx, query, releaseName).Scan(&metadata)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, storage.ErrNotExist
		}
		return nil, newQueryError(query, err, releaseName)
	}

	return metadata, nil
}
