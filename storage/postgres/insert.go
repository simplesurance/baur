package postgres

import (
	"database/sql"
	"fmt"

	"github.com/pkg/errors"

	"github.com/simplesurance/baur/storage"
)

func insertBuild(tx *sql.Tx, appID, vcsID int, b *storage.Build) (int, error) {
	const stmt = `
	INSERT INTO build
	(application_id, vcs_id, start_timestamp, stop_timestamp, total_input_digest)
	VALUES($1, $2, $3, $4, $5)
	RETURNING id;`

	var id int

	r := tx.QueryRow(stmt, appID, vcsID, b.StartTimeStamp, b.StopTimeStamp, b.TotalInputDigest)

	if err := r.Scan(&id); err != nil {
		return -1, err
	}

	return id, nil
}

func insertBuildOutputs(tx *sql.Tx, buildID int, outputIDs []int) ([]int, error) {
	const stmt1 = "INSERT INTO build_output(build_id, output_id) VALUES"
	const stmt2 = "RETURNING ID"

	var ids []int
	var stmtVals string

	for i, outputID := range outputIDs {
		stmtVals += fmt.Sprintf("(%d, %d)", buildID, outputID)

		if i < len(outputIDs)-1 {
			stmtVals += ", "
		}
	}

	query := stmt1 + stmtVals + stmt2
	rows, err := tx.Query(query)
	if err != nil {
		return nil, errors.Wrapf(err, "db query %q failed", query)
	}

	for rows.Next() {
		var id int

		err := rows.Scan(&id)
		if err != nil {
			rows.Close()
			return nil, errors.Wrapf(err, "parsing result of query %q failed", query)
		}

		ids = append(ids, id)
	}

	if err := rows.Err(); err != nil {
		return nil, errors.Wrap(err, "iterating over rows failed")
	}

	return ids, nil
}

func insertOutputsIfNotExist(tx *sql.Tx, outputs []*storage.Output) ([]int, error) {
	const stmt1 = "INSERT INTO output (name, type, digest, size_bytes) VALUES"
	const stmt2 = `
	ON CONFLICT ON CONSTRAINT output_digest_key
	DO UPDATE SET id=output.id RETURNING id
	`

	var (
		argCNT    = 1
		stmtVals  string
		queryArgs = make([]interface{}, 0, len(outputs)*4)
		ids       = make([]int, 0, len(outputs))
	)

	for i, out := range outputs {
		stmtVals += fmt.Sprintf("($%d, $%d, $%d, $%d)", argCNT, argCNT+1, argCNT+2, argCNT+3)
		argCNT += 4
		queryArgs = append(queryArgs, out.Name, out.Type, out.Digest, out.SizeBytes)

		if i < len(outputs)-1 {
			stmtVals += ", "
		}
	}
	query := stmt1 + stmtVals + stmt2

	rows, err := tx.Query(query, queryArgs...)
	if err != nil {
		return nil, errors.Wrapf(err, "db query %q failed", query)
	}

	for rows.Next() {
		var id int

		err := rows.Scan(&id)
		if err != nil {
			rows.Close()
			return nil, errors.Wrapf(err, "parsing result of query %q failed", query)
		}

		ids = append(ids, id)
	}

	if err := rows.Err(); err != nil {
		return nil, errors.Wrap(err, "iterating over rows failed")
	}

	return ids, nil
}

func insertInputBuilds(tx *sql.Tx, buildID int, inputIDs []int) error {
	const stmt1 = `
		INSERT into input_build 
		(build_id, input_id)
		VALUES
	`

	var stmtVals string
	argCNT := 1
	queryArgs := make([]interface{}, 0, len(inputIDs))

	queryArgs = append(queryArgs, buildID)
	argCNT++

	for i, id := range inputIDs {
		stmtVals += fmt.Sprintf("($1, $%d)", argCNT)
		argCNT++

		queryArgs = append(queryArgs, id)

		if i < len(inputIDs)-1 {
			stmtVals += ", "
		}
	}

	query := stmt1 + stmtVals

	_, err := tx.Exec(query, queryArgs...)
	if err != nil {
		return errors.Wrapf(err, "db query %q failed", query)
	}

	return nil
}

func insertInputsIfNotExist(tx *sql.Tx, inputs []*storage.Input) ([]int, error) {
	const stmt1 = "INSERT INTO input (uri, type, digest) VALUES"
	const stmt2 = `
	ON CONFLICT ON CONSTRAINT input_uniq
	DO UPDATE SET id=input.id RETURNING id
	`
	var (
		stmtVals string

		argCNT    = 1
		queryArgs = make([]interface{}, 0, len(inputs)*2)
		ids       = make([]int, 0, len(inputs))
	)

	for i, in := range inputs {
		stmtVals += fmt.Sprintf("($%d, $%d, $%d)", argCNT, argCNT+1, argCNT+2)
		argCNT += 3
		queryArgs = append(queryArgs, in.URI, in.Type, in.Digest)

		if i < len(inputs)-1 {
			stmtVals += ", "
		}
	}

	query := stmt1 + stmtVals + stmt2

	rows, err := tx.Query(query, queryArgs...)
	if err != nil {
		return nil, errors.Wrapf(err, "db query %q failed", query)
	}

	for rows.Next() {
		var id int

		err := rows.Scan(&id)
		if err != nil {
			rows.Close()
			return nil, errors.Wrapf(err, "db query %q failed", query)
		}

		ids = append(ids, id)
	}

	if err := rows.Err(); err != nil {
		return nil, errors.Wrap(err, "iterating over rows failed")
	}

	return ids, nil

}

func insertVCSIfNotExist(tx *sql.Tx, v *storage.VCSState) (int, error) {
	const stmt = `
	INSERT INTO vcs
	(commit, dirty)
	VALUES($1, $2)
	ON CONFLICT ON CONSTRAINT vcs_uniq
	DO UPDATE SET id=vcs.id RETURNING id
	`
	var id int

	err := tx.QueryRow(stmt, v.CommitID, v.IsDirty).Scan(&id)
	if err != nil {
		return -1, errors.Wrapf(err, "db query %q failed", stmt)
	}

	return id, nil
}

func insertAppIfNotExist(tx *sql.Tx, app *storage.Application) error {
	const stmt = `
	INSERT INTO application
	(name)
	VALUES($1)
	ON CONFLICT ON CONSTRAINT application_name_key
	DO UPDATE SET id=application.id RETURNING id
	`

	err := tx.QueryRow(stmt, app.NameLower()).Scan(&app.ID)
	if err != nil {
		return errors.Wrapf(err, "db query %q failed", stmt)
	}

	return nil
}

func insertUploads(tx *sql.Tx, buildOutputIDs []int, outputs []*storage.Output) error {
	const stmt = `
	INSERT into upload
	(build_output_id, uri, upload_duration_ns)
	VALUES
	`

	var (
		stmtVals  string
		argCNT    = 1
		queryArgs = make([]interface{}, 0, len(outputs)*4)
	)

	// TODO: retrieve the ID from the insert and set it in out.Upload
	if len(outputs) != len(buildOutputIDs) {
		return fmt.Errorf("output (%d) and buildOutputIDs (%d) slices are not of same length",
			len(outputs), len(buildOutputIDs))
	}

	for i, out := range outputs {
		stmtVals += fmt.Sprintf("($%d, $%d, $%d)", argCNT, argCNT+1, argCNT+2)
		argCNT += 3
		queryArgs = append(queryArgs, buildOutputIDs[i], out.Upload.URI, out.Upload.UploadDuration)

		if i < len(outputs)-1 {
			stmtVals += ", "
		}
	}

	query := stmt + stmtVals

	_, err := tx.Exec(query, queryArgs...)
	if err != nil {
		return errors.Wrapf(err, "db query %q failed", query)
	}

	return err
}

// Save stores a build in the database, the ID field of the passed Build is
// ignored. The database generates a record ID and it will be stored in the
// passed Build.
func (c *Client) Save(b *storage.Build) error {
	tx, err := c.Db.Begin()
	if err != nil {
		return errors.Wrap(err, "starting transaction failed")
	}

	defer func() {
		if err != nil {
			tx.Rollback()
		} else {
			tx.Commit()
		}
	}()

	err = insertAppIfNotExist(tx, &b.Application)
	if err != nil {
		return errors.Wrap(err, "storing application record failed")
	}

	vcsID, err := insertVCSIfNotExist(tx, &b.VCSState)
	if err != nil {
		return errors.Wrap(err, "storing vcs information failed")
	}

	buildID, err := insertBuild(tx, b.Application.ID, vcsID, b)
	if err != nil {
		return errors.Wrap(err, "storing build record failed")
	}

	outputIDs, err := insertOutputsIfNotExist(tx, b.Outputs)
	if err != nil {
		return errors.Wrap(err, "storing output records failed")
	}

	buildOutputIDs, err := insertBuildOutputs(tx, buildID, outputIDs)
	if err != nil {
		return errors.Wrap(err, "storing buildOutput records failed")
	}

	err = insertUploads(tx, buildOutputIDs, b.Outputs)
	if err != nil {
		return errors.Wrap(err, "storing upload record failed")
	}

	// inputs not specified in the baur app config
	if len(b.Inputs) == 0 {
		return nil
	}

	ids, err := insertInputsIfNotExist(tx, b.Inputs)
	if err != nil {
		return errors.Wrap(err, "storing inputs failed")
	}

	err = insertInputBuilds(tx, buildID, ids)
	if err != nil {
		return errors.Wrap(err, "storing input_build failed")
	}

	b.ID = buildID

	return nil
}
