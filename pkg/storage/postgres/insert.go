package postgres

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/jackc/pgx/v4"

	"github.com/simplesurance/baur/v2/pkg/storage"
)

func strArgList(args ...interface{}) string {
	var result strings.Builder

	result.WriteRune('[')

	for i, arg := range args {
		fmt.Fprintf(&result, "'%v'", arg)

		if i < len(args)-1 {
			result.WriteString(", ")
		}
	}

	result.WriteRune(']')

	return result.String()
}

// queryValueStr returns the argument for an SQL VALUES statement with
// enumerated parameters.
// It creates pairsCount "($n, $n+1, $n+...)" string pairs, with argsPerPair
// values per pair.
func queryValueStr(pairsCount, argsPerPair int) string {
	var res strings.Builder

	// allocation size is not exact but better then no preallocation:
	// 4 Bytes per parameter '$nn,' +
	// 4 bytes for the opening bracket, closing bracket, commata and space
	res.Grow((argsPerPair * 4) + (pairsCount * 3))

	argNr := 1
	for i := 0; i < pairsCount; i++ {
		if i > 0 {
			res.WriteRune(' ')
		}

		res.WriteRune('(')

		for j := 0; j < argsPerPair; j++ {
			fmt.Fprintf(&res, "$%d", argNr)
			argNr++

			if j < argsPerPair-1 {
				res.WriteString(", ")
			}
		}

		res.WriteRune(')')

		if i < pairsCount-1 {
			res.WriteString(", ")
		}
	}

	return res.String()
}

func scanIDs(rows pgx.Rows, res *[]int) error {
	for rows.Next() {
		var id int

		err := rows.Scan(&id)
		if err != nil {
			rows.Close()
			return err
		}

		*res = append(*res, id)
	}

	return rows.Err()
}

func insertAppIfNotExist(ctx context.Context, db dbConn, appName string) (int, error) {
	const query = `
	   INSERT INTO application (name)
	   VALUES ($1)
	       ON CONFLICT ON CONSTRAINT application_name_uniq
	       DO UPDATE SET id=application.id
	RETURNING id
	`

	var id int

	if err := db.QueryRow(ctx, query, appName).Scan(&id); err != nil {
		return -1, newQueryError(query, err, appName)
	}

	return id, nil
}

func insertTaskIfNotExist(ctx context.Context, db dbConn, appName, taskName string) (int, error) {
	var id int

	appID, err := insertAppIfNotExist(ctx, db, appName)
	if err != nil {
		return -1, err
	}

	const query = `
	   INSERT INTO task (name, application_id)
	   VALUES ($1, $2)
	       ON CONFLICT ON CONSTRAINT task_name_application_id_uniq
	       DO UPDATE SET id=task.id
	RETURNING id
	`

	if err := db.QueryRow(ctx, query, taskName, appID).Scan(&id); err != nil {
		return -1, newQueryError(query, err, appName, taskName)
	}

	return id, nil
}

func insertVCSIfNotExist(ctx context.Context, db dbConn, revision string, isDirty bool) (int, error) {
	const query = `
	   INSERT INTO vcs (revision, dirty)
	   VALUES ($1, $2)
	       ON CONFLICT ON CONSTRAINT vcs_revision_dirty_uniq
	       DO UPDATE SET id=vcs.id
	RETURNING id
	`

	var id int

	if err := db.QueryRow(ctx, query, revision, isDirty).Scan(&id); err != nil {
		return -1, newQueryError(query, err, revision, isDirty)
	}

	return id, nil
}

// clonedSortedInputfiles returns a shallow-copied sorted variant of inputs.
func clonedSortedInputfiles(inputs []*storage.InputFile) []*storage.InputFile {
	inputs = append([]*storage.InputFile{}, inputs...)

	sort.Slice(inputs, func(i, j int) bool {
		switch strings.Compare(inputs[i].Path, inputs[j].Path) {
		case -1:
			return true
		case 1:
			return false
		}

		return strings.Compare(inputs[i].Digest, inputs[j].Digest) == -1
	})

	return inputs
}

// clonedSortedInputStrings returns a shallow-copied sorted variant of inputs.
func clonedSortedInputStrings(inputs []*storage.InputString) []*storage.InputString {
	inputs = append([]*storage.InputString{}, inputs...)

	sort.Slice(inputs, func(i, j int) bool {
		switch strings.Compare(inputs[i].String, inputs[j].String) {
		case -1:
			return true
		case 1:
			return false
		}

		return strings.Compare(inputs[i].Digest, inputs[j].Digest) == -1
	})

	return inputs
}

func insertInputFilesIfNotExist(ctx context.Context, db dbConn, inputs []*storage.InputFile) ([]int, error) {
	const stmt1 = `
           INSERT INTO input_file (path, digest)
	   VALUES
`
	const stmt2 = `
	       ON CONFLICT ON CONSTRAINT input_file_path_digest_uniq
	       DO UPDATE SET id=input_file.id
	RETURNING id
	`

	// inputs are sorted to prevent an deadlock when running multiple
	// transaction in parallel doing inserts, see
	// https://github.com/simplesurance/baur/issues/343
	inputs = clonedSortedInputfiles(inputs)

	stmtVals := queryValueStr(len(inputs), 2)

	queryArgs := make([]interface{}, 0, len(inputs)*2)
	for _, in := range inputs {
		queryArgs = append(queryArgs, in.Path, in.Digest)
	}

	query := stmt1 + stmtVals + " " + stmt2

	rows, err := db.Query(ctx, query, queryArgs...)
	if err != nil {
		return nil, newQueryError(query, err, queryArgs...)
	}

	ids := make([]int, 0, len(inputs))
	if err := scanIDs(rows, &ids); err != nil {
		return nil, newQueryError(query, err, queryArgs...)
	}

	return ids, nil
}

func insertInputStringFilesIfNotExist(ctx context.Context, db dbConn, inputs []*storage.InputString) ([]int, error) {
	const stmt1 = `
           INSERT INTO input_string (string, digest)
	   VALUES
`
	const stmt2 = `
	       ON CONFLICT ON CONSTRAINT input_string_digest_uniq
	       DO UPDATE SET id=input_string.id
	RETURNING id
	`

	// inputs are sorted to prevent an deadlock when running multiple
	// transaction in parallel doing inserts, see
	// https://github.com/simplesurance/baur/issues/343
	inputs = clonedSortedInputStrings(inputs)

	stmtVals := queryValueStr(len(inputs), 2)

	queryArgs := make([]interface{}, 0, len(inputs)*2)
	for _, in := range inputs {
		queryArgs = append(queryArgs, in.String, in.Digest)
	}

	query := stmt1 + stmtVals + " " + stmt2

	rows, err := db.Query(ctx, query, queryArgs...)
	if err != nil {
		return nil, newQueryError(query, err, queryArgs...)
	}

	ids := make([]int, 0, len(inputs))
	if err := scanIDs(rows, &ids); err != nil {
		return nil, newQueryError(query, err, queryArgs...)
	}

	return ids, nil
}

func insertTaskRunInputStringsIfNotExist(ctx context.Context, db dbConn, taskRunID int, inputStrings []*storage.InputString) error {
	const stmt1 = `
	INSERT INTO task_run_string_input (task_run_id, input_string_id)
	VALUES
	`

	if len(inputStrings) == 0 {
		return nil
	}

	inputStringIDs, err := insertInputStringFilesIfNotExist(ctx, db, inputStrings)
	if err != nil {
		return err
	}

	var stmtVals strings.Builder
	argNr := 2
	for i := 0; i < len(inputStringIDs); i++ {
		fmt.Fprintf(&stmtVals, "($1, $%d)", argNr)
		argNr++

		if i < len(inputStringIDs)-1 {
			stmtVals.WriteString(", ")
		}
	}

	queryArgs := make([]interface{}, 1, (len(inputStringIDs)*2)+2)
	queryArgs[0] = taskRunID

	for _, inputID := range inputStringIDs {
		queryArgs = append(queryArgs, inputID)
	}

	query := stmt1 + stmtVals.String()

	_, err = db.Exec(ctx, query, queryArgs...)
	if err != nil {
		return newQueryError(query, err, queryArgs...)
	}

	return nil
}

func insertTaskRunInputFilesIfNotExist(ctx context.Context, db dbConn, taskRunID int, inputFiles []*storage.InputFile) error {
	const stmt1 = `
	INSERT INTO task_run_file_input (task_run_id, input_file_id)
	VALUES
	`

	if len(inputFiles) == 0 {
		return nil
	}

	inputIDs, err := insertInputFilesIfNotExist(ctx, db, inputFiles)
	if err != nil {
		return err
	}

	var stmtVals strings.Builder
	argNr := 2
	for i := 0; i < len(inputIDs); i++ {
		fmt.Fprintf(&stmtVals, "($1, $%d)", argNr)
		argNr++

		if i < len(inputIDs)-1 {
			stmtVals.WriteString(", ")
		}
	}

	queryArgs := make([]interface{}, 1, (len(inputIDs)*2)+2)
	queryArgs[0] = taskRunID

	for _, inputID := range inputIDs {
		queryArgs = append(queryArgs, inputID)
	}

	query := stmt1 + stmtVals.String()

	_, err = db.Exec(ctx, query, queryArgs...)
	if err != nil {
		return newQueryError(query, err, queryArgs...)
	}

	return nil
}

func insertUploads(ctx context.Context, db dbConn, uploads []*storage.Upload) ([]int, error) {
	const stmt1 = `
	INSERT into upload (uri, method, start_timestamp, stop_timestamp)
	VALUES`
	const stmt2 = "RETURNING id"

	stmtVals := queryValueStr(len(uploads), 4)

	queryArgs := make([]interface{}, 0, len(uploads)*4)
	for _, upload := range uploads {
		queryArgs = append(
			queryArgs,
			upload.URI, upload.Method, upload.UploadStartTimestamp, upload.UploadStopTimestamp,
		)
	}

	query := stmt1 + stmtVals + " " + stmt2

	rows, err := db.Query(ctx, query, queryArgs...)
	if err != nil {
		return nil, newQueryError(query, err, queryArgs...)
	}

	ids := make([]int, 0, len(uploads))
	if err := scanIDs(rows, &ids); err != nil {
		return nil, newQueryError(query, err, queryArgs...)
	}

	return ids, err
}

func insertOutputIfNotExist(ctx context.Context, db dbConn, output *storage.Output) (int, error) {
	const query = `
	   INSERT INTO output (name, type, digest, size_bytes)
	   VALUES($1, $2, $3, $4)
	       ON CONFLICT ON CONSTRAINT output_name_type_digest_size_bytes_uniq
	       DO UPDATE SET id=output.id
	RETURNING id
	`

	var id int

	queryArgs := []interface{}{
		output.Name,
		output.Type,
		output.Digest,
		output.SizeBytes,
	}

	err := db.QueryRow(
		ctx,
		query,
		queryArgs...,
	).Scan(&id)
	if err != nil {
		return -1, newQueryError(query, err, queryArgs...)
	}

	return id, nil
}

func insertTaskOutputsIfNotExist(ctx context.Context, db dbConn, taskRunID int, outputs []*storage.Output) error {
	if len(outputs) == 0 {
		return nil
	}

	type taskOutput struct {
		outputID int
		uploadID int
	}

	var records []*taskOutput

	for _, output := range outputs {
		outputID, err := insertOutputIfNotExist(ctx, db, output)
		if err != nil {
			return err
		}

		uploadIDs, err := insertUploads(ctx, db, output.Uploads)
		if err != nil {
			return err
		}

		for _, uploadID := range uploadIDs {
			records = append(records, &taskOutput{
				outputID: outputID,
				uploadID: uploadID,
			})
		}
	}

	const stmt1 = "INSERT INTO task_run_output (task_run_id, output_id, upload_id) VALUES"

	stmtVals := queryValueStr(len(records), 3)

	queryArgs := make([]interface{}, 0, len(records)*3)
	for _, record := range records {
		queryArgs = append(queryArgs, taskRunID, record.outputID, record.uploadID)
	}

	query := stmt1 + stmtVals

	_, err := db.Exec(ctx, query, queryArgs...)
	if err != nil {
		return newQueryError(query, err, queryArgs...)
	}

	return nil
}

func (c *Client) saveTaskRun(ctx context.Context, tx pgx.Tx, taskRun *storage.TaskRunFull) (int, error) {
	const query = `
		   INSERT INTO task_run (vcs_id, task_id, total_input_digest, start_timestamp, stop_timestamp, result)
		   VALUES($1, $2, $3, $4, $5, $6)
		RETURNING ID
		`

	var taskRunID int

	vcsID, err := insertVCSIfNotExist(ctx, tx, taskRun.VCSRevision, taskRun.VCSIsDirty)
	if err != nil {
		return -1, fmt.Errorf("storing vcs record failed: %w", err)
	}

	taskID, err := insertTaskIfNotExist(ctx, tx, taskRun.ApplicationName, taskRun.TaskName)
	if err != nil {
		return -1, fmt.Errorf("storing task record failed: %w", err)
	}

	queryArgs := []interface{}{
		vcsID,
		taskID,
		taskRun.TotalInputDigest,
		taskRun.StartTimestamp,
		taskRun.StopTimestamp,
		taskRun.Result,
	}

	err = tx.QueryRow(
		ctx,
		query,
		queryArgs...,
	).Scan(&taskRunID)
	if err != nil {
		return -1, newQueryError(query, err, queryArgs...)
	}

	err = insertTaskRunInputStringsIfNotExist(ctx, tx, taskRunID, taskRun.Inputs.Strings)
	if err != nil {
		return -1, err
	}

	err = insertTaskRunInputFilesIfNotExist(ctx, tx, taskRunID, taskRun.Inputs.Files)
	if err != nil {
		return -1, err
	}

	err = insertTaskOutputsIfNotExist(ctx, tx, taskRunID, taskRun.Outputs)
	if err != nil {
		return -1, err
	}

	return taskRunID, nil
}

func (c *Client) SaveTaskRun(ctx context.Context, taskRun *storage.TaskRunFull) (int, error) {
	tx, err := c.db.Begin(ctx)
	if err != nil {
		return -1, err
	}

	id, err := c.saveTaskRun(ctx, tx, taskRun)
	if err != nil {
		_ = tx.Rollback(ctx)
		return -1, err
	}

	if err := tx.Commit(ctx); err != nil {
		return -1, err
	}

	return id, nil
}
