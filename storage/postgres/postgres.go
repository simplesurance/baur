package postgres

import (
	"database/sql"
	"fmt"
	"strings"
	"time"

	_ "github.com/lib/pq" // postgresql
	"github.com/pkg/errors"

	"github.com/simplesurance/baur/storage"
)

// Client is a postgres storage client
type Client struct {
	Db *sql.DB
}

type SqlFields map[storage.Field]string

type SqlFilterOperators map[storage.SortOperator]string

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
		Db: db,
	}, nil
}

// Close closes the connection
func (c *Client) Close() {
	c.Db.Close()
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
	(build_output_id, url, upload_duration_ns)
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
		queryArgs = append(queryArgs, buildOutputIDs[i], out.Upload.URL, out.Upload.UploadDuration)

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

func (c *Client) GetBuildOutputs(buildId int) ([]*storage.Output, error) {
	const stmt = `SELECT
			output.name, output.digest, output.type, output.size_bytes,
			upload.id, upload.url, upload.upload_duration_ns
		      FROM output
		      JOIN build_output ON output.id = build_output.output_id
		      JOIN upload ON upload.build_output_id = build_output.id
		      WHERE build_output.build_id = $1
		      `

	rows, err := c.Db.Query(stmt, buildId)
	if err != nil {
		return nil, errors.Wrapf(err, "db query %q failed", stmt)
	}

	var outputs []*storage.Output

	for rows.Next() {
		var output storage.Output

		err := rows.Scan(
			&output.Name,
			&output.Digest,
			&output.Type,
			&output.SizeBytes,
			&output.Upload.ID,
			&output.Upload.URL,
			&output.Upload.UploadDuration,
		)
		if err != nil {
			return nil, errors.Wrapf(err, "db query %q failed", stmt)
		}

		outputs = append(outputs, &output)
	}

	if err := rows.Err(); err != nil {
		return nil, errors.Wrap(err, "iterating over rows failed")
	}

	return outputs, nil
}

func (c *Client) populateOutputs(build *storage.Build) error {
	const stmt = `SELECT
			output.name, output.digest, output.type, output.size_bytes,
			upload.id, upload.url, upload.upload_duration_ns
		      FROM output
		      JOIN build_output ON output.id = build_output.output_id
		      JOIN upload ON upload.build_output_id = build_output.id
		      WHERE build_output.build_id = $1
		      `

	rows, err := c.Db.Query(stmt, build.ID)
	if err != nil {
		return errors.Wrapf(err, "db query %q failed", stmt)
	}

	for rows.Next() {
		var output storage.Output

		err := rows.Scan(
			&output.Name,
			&output.Digest,
			&output.Type,
			&output.SizeBytes,
			&output.Upload.ID,
			&output.Upload.URL,
			&output.Upload.UploadDuration,
		)
		if err != nil {
			return errors.Wrapf(err, "db query %q failed", stmt)
		}

		build.Outputs = append(build.Outputs, &output)
	}

	if err := rows.Err(); err != nil {
		return errors.Wrap(err, "iterating over rows failed")
	}

	return nil
}

// GetBuildWithoutInputs retrieves a build from the database
func (c *Client) GetBuildWithoutInputs(id int) (*storage.Build, error) {
	builds, err := c.GetBuildsWithoutInputs([]int{id})
	if err != nil {
		return nil, err
	}

	// should not happen, GetBuildsWithoutInputs should return an error if
	// no records are found, one id should only match one db id
	if len(builds) != 1 {
		panic(fmt.Sprintf("postgres: GetBuildsWithoutInputs returned no error and %d results when querying for 1 id", len(builds)))
	}

	return builds[0], nil
}

// GetLatestBuildByDigest returns the build id of a build for the application
// with the passed digest. If multiple builds exist, the one with the lastest
// stop_timestamp is returned.
// Inputs are not fetched from the database.
// If no builds exist sql.ErrNoRows is returned
func (c *Client) GetLatestBuildByDigest(appName, totalInputDigest string) (*storage.Build, error) {
	var (
		build    storage.Build
		commitID sql.NullString
		isDirty  sql.NullBool
	)

	const stmt = `
	 SELECT app.id, app.name,
		build.id, build.start_timestamp, build.stop_timestamp, build.total_input_digest,
		vcs.commit, vcs.dirty
	 FROM application AS app
	 JOIN build ON app.id = build.application_id
	 LEFT OUTER JOIN vcs ON vcs.id = build.vcs_id
	 WHERE app.name = $1 AND total_input_digest = $2
	 ORDER BY build.stop_timestamp DESC LIMIT 1
	 `

	err := c.Db.QueryRow(stmt, appName, totalInputDigest).Scan(
		&build.Application.ID,
		&build.Application.Name,
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

	builds, err := c.GetBuildOutputs(build.ID)
	if err != nil {
		return nil, errors.Wrap(err, "fetching build outputs failed")
	}

	build.Outputs = builds

	return &build, err
}

// GetApps returns all application records ordered by Name
func (c *Client) GetApps() ([]*storage.Application, error) {
	const query = "SELECT id, name FROM application ORDER BY name"
	var res []*storage.Application

	rows, err := c.Db.Query(query)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, storage.ErrNotExist
		}

		return nil, errors.Wrapf(err, "db query %q failed", query)
	}

	for rows.Next() {
		var app storage.Application

		err := rows.Scan(&app.ID, &app.Name)
		if err != nil {
			rows.Close()
			return nil, errors.Wrapf(err, "parsing result of query %q failed", query)
		}

		res = append(res, &app)
	}

	if err := rows.Err(); err != nil {
		return nil, errors.Wrap(err, "iterating over rows failed")
	}

	return res, nil
}

// GetSameTotalInputDigestsForAppBuilds finds TotalInputDigests that are the
// same for builds of an app with a build start time not before startTs
// If not builds with the same totalInputDigest is found, an empty slice is
// returned.
func (c *Client) GetSameTotalInputDigestsForAppBuilds(appName string, startTs time.Time) (map[string][]int, error) {
	const query = `
		 WITH data AS(
			 SELECT total_input_digest from build
			 JOIN application on build.application_id = application.id
			 WHERE total_input_digest != ''
			 AND build.start_timestamp  >= $1
			 AND application.name = $2
			 GROUP BY total_input_digest
			 HAVING count(total_input_digest) > 1)

		SELECT build.id, build.total_input_digest FROM data
		JOIN build ON build.total_input_digest = data.total_input_digest
		JOIN application on build.application_id = application.id
		WHERE build.start_timestamp  >= $1
		AND application.name = $2`

	res := map[string][]int{}

	rows, err := c.Db.Query(query, startTs, appName)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, storage.ErrNotExist
		}

		return nil, errors.Wrapf(err, "db query %q failed", query)
	}

	for rows.Next() {
		var digest string
		var buildID int

		err := rows.Scan(&buildID, &digest)
		if err != nil {
			rows.Close()
			return nil, errors.Wrapf(err, "parsing result of query %q failed", query)
		}

		res[digest] = append(res[digest], buildID)
	}

	if err := rows.Err(); err != nil {
		return nil, errors.Wrap(err, "iterating over rows failed")
	}

	return res, err
}

func toStringSlice(ints []int) []string {
	res := make([]string, 0, len(ints))

	for _, val := range ints {
		res = append(res, fmt.Sprint(val))
	}

	return res
}

// GetBuildsWithoutInputs returns multiple builds by their IDs
func (c *Client) GetBuildsWithoutInputs(buildIDs []int) ([]*storage.Build, error) {
	query := fmt.Sprintf(`
		SELECT app.id, app.name,
		       build.id, build.start_timestamp, build.stop_timestamp, build.total_input_digest,
		       vcs.commit, vcs.dirty
		FROM application AS app
		JOIN build ON app.id = build.application_id
		LEFT OUTER JOIN vcs ON vcs.id = build.vcs_id
		WHERE build.id IN (%s)`, strings.Join(toStringSlice(buildIDs), ", "))

	var res []*storage.Build

	rows, err := c.Db.Query(query)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, storage.ErrNotExist
		}

		return nil, errors.Wrapf(err, "db query %q failed", query)
	}

	for rows.Next() {
		var build storage.Build
		var commitID sql.NullString
		var isDirty sql.NullBool

		err := rows.Scan(
			&build.Application.ID,
			&build.Application.Name,
			&build.ID,
			&build.StartTimeStamp,
			&build.StopTimeStamp,
			&build.TotalInputDigest,
			&commitID,
			&isDirty)
		if err != nil {
			rows.Close()
			return nil, errors.Wrapf(err, "parsing result of query %q failed", query)
		}

		if commitID.Valid {
			build.VCSState.CommitID = commitID.String
			build.VCSState.IsDirty = isDirty.Bool
		}

		if err := c.populateOutputs(&build); err != nil {
			return nil, errors.Wrap(err, "fetching build outputs failed")
		}

		res = append(res, &build)
	}

	if err := rows.Err(); err != nil {
		return nil, errors.Wrap(err, "iterating over rows failed")
	}

	if len(res) == 0 {
		return nil, storage.ErrNotExist
	}

	return res, err
}
