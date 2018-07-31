package postgres

import (
	"database/sql"
	"fmt"

	_ "github.com/lib/pq" // postgresql
	"github.com/pkg/errors"

	"github.com/simplesurance/baur/storage"
)

// Client is a postgres storage client
type Client struct {
	db *sql.DB
}

// New establishes a connection a postgres db
func New(url string) (*Client, error) {
	db, err := sql.Open("postgres", url)
	if err != nil {
		return nil, err
	}

	if err := db.Ping(); err != nil {
		return nil, err
	}

	return &Client{
		db: db,
	}, nil
}

// Close closes the connection
func (c *Client) Close() {
	c.db.Close()
}

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
	const stmt1 = "INSERT INTO input (url, digest) VALUES"
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
		stmtVals += fmt.Sprintf("($%d, $%d)", argCNT, argCNT+1)
		argCNT += 2
		queryArgs = append(queryArgs, in.URL, in.Digest)

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

func insertAppIfNotExist(tx *sql.Tx, appName string) (int, error) {
	const stmt = `
	INSERT INTO application
	(name)
	VALUES($1)
	ON CONFLICT ON CONSTRAINT application_name_key
	DO UPDATE SET id=application.id RETURNING id
	`
	var id int

	err := tx.QueryRow(stmt, appName).Scan(&id)
	if err != nil {
		return -1, errors.Wrapf(err, "db query %q failed", stmt)
	}

	return id, nil
}

func insertUploads(tx *sql.Tx, buildOutputIDs []int, outputs []*storage.Output) error {
	const stmt = `
	INSERT into upload
	(build_output_id, url, upload_duration_ns)
	VALUES
	`

	var (
		stmtVals  string
		argCNT    = 1
		queryArgs = make([]interface{}, 0, len(outputs)*4)
	)

	if len(outputs) != len(buildOutputIDs) {
		return fmt.Errorf("output (%d) and buildOutputIDs (%d) slices are not of same length",
			len(outputs), len(buildOutputIDs))
	}

	for i, out := range outputs {
		stmtVals += fmt.Sprintf("($%d, $%d, $%d)", argCNT, argCNT+1, argCNT+2)
		argCNT += 3
		queryArgs = append(queryArgs, buildOutputIDs[i], out.URL, out.UploadDuration)

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
	tx, err := c.db.Begin()
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

	appID, err := insertAppIfNotExist(tx, b.AppNameLower())
	if err != nil {
		return errors.Wrap(err, "storing application record failed")
	}

	vcsID, err := insertVCSIfNotExist(tx, &b.VCSState)
	if err != nil {
		return errors.Wrap(err, "storing vcs information failed")
	}

	buildID, err := insertBuild(tx, appID, vcsID, b)
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

func (c *Client) populateOutputs(build *storage.Build) error {
	const stmt = `SELECT
			output.name, output.digest, output.type, output.size_bytes,
			upload.url, upload.upload_duration_ns
		      FROM output
		      JOIN build_output ON output.id = build_output.output_id
		      JOIN upload ON upload.build_output_id = build_output.id
		      WHERE build_output.build_id = $1
		      `

	rows, err := c.db.Query(stmt, build.ID)
	if err != nil {
		return errors.Wrapf(err, "db query %q failed", stmt)
	}

	for rows.Next() {
		var output storage.Output

		rows.Scan(
			&output.Name,
			&output.Digest,
			&output.Type,
			&output.SizeBytes,
			&output.URL,
			&output.UploadDuration,
		)

		build.Outputs = append(build.Outputs, &output)
	}

	if err := rows.Err(); err != nil {
		return errors.Wrap(err, "iterating over rows failed")
	}

	return nil
}

// GetBuildWithoutInputs retrieves a build from the database
func (c *Client) GetBuildWithoutInputs(id int) (*storage.Build, error) {
	var commitID sql.NullString
	var isDirty sql.NullBool
	build := storage.Build{ID: id}

	const stmt = `
	 SELECT app.name,
		build.start_timestamp, build.stop_timestamp, build.total_input_digest,
		vcs.commit, vcs.dirty
	 FROM application AS app
	 JOIN build ON app.id = build.application_id
	 LEFT OUTER JOIN vcs ON vcs.id = build.vcs_id
	 WHERE build.id = $1`

	err := c.db.QueryRow(stmt, build.ID).Scan(
		&build.AppName,
		&build.StartTimeStamp,
		&build.StopTimeStamp,
		&build.TotalInputDigest,
		&commitID,
		&isDirty)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, storage.ErrNotExist
		}

		return nil, errors.Wrapf(err, "db query %q failed", stmt)
	}

	if commitID.Valid {
		build.VCSState.CommitID = commitID.String
	}

	build.VCSState.IsDirty = isDirty.Bool

	if err := c.populateOutputs(&build); err != nil {
		return nil, errors.Wrap(err, "fetching build outputs failed")
	}

	return &build, err
}

// GetLatestBuildByDigest returns the build id of a build for the
// application with the passed digest. If multiple builds exist, the one with
// the lastest stop_timestamp is returned.
// If no builds exist sql.ErrNoRows is returned
func (c *Client) GetLatestBuildByDigest(appName, totalInputDigest string) (*storage.Build, error) {
	var (
		build    storage.Build
		commitID sql.NullString
		isDirty  sql.NullBool
	)

	const stmt = `
	 SELECT app.name, build.id,
		build.start_timestamp, build.stop_timestamp, build.total_input_digest,
		vcs.commit, vcs.dirty
	 FROM application AS app
	 JOIN build ON app.id = build.application_id
	 LEFT OUTER JOIN vcs ON vcs.id = build.vcs_id
	 WHERE app.name = $1 AND total_input_digest = $2
	 ORDER BY build.stop_timestamp DESC LIMIT 1
	 `

	err := c.db.QueryRow(stmt, appName, totalInputDigest).Scan(
		&build.AppName,
		&build.ID,
		&build.StartTimeStamp,
		&build.StopTimeStamp,
		&build.TotalInputDigest,
		&commitID,
		&isDirty)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, storage.ErrNotExist
		}

		return nil, errors.Wrapf(err, "db query %q failed", stmt)
	}

	if commitID.Valid {
		build.VCSState.CommitID = commitID.String
		build.VCSState.IsDirty = isDirty.Bool
	}

	if err := c.populateOutputs(&build); err != nil {
		return nil, errors.Wrap(err, "fetching build outputs failed")
	}

	return &build, err
}
