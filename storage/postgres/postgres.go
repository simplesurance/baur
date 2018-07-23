package postgres

import (
	"database/sql"
	"time"

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

// ListBuildsPerApp returns all builds for an app
func (c *Client) ListBuildsPerApp(appName string, maxResults int) ([]*storage.Build, error) {
	return nil, nil
}

func insertBuild(tx *sql.Tx, appID int, b *storage.Build) (int, error) {
	const stmt = `
	INSERT INTO build
	(application_id, start_timestamp, stop_timestamp, total_input_digest)
	VALUES($1, $2, $3, $4)
	RETURNING id;`

	var id int

	r := tx.QueryRow(stmt, appID, b.StartTimeStamp, b.StopTimeStamp, b.TotalInputDigest)

	if err := r.Scan(&id); err != nil {
		return -1, err
	}

	return id, nil
}

func insertOutputIfNotExist(tx *sql.Tx, a *storage.Output) (int, error) {
	const stmt = `
	INSERT INTO output
	(name, type, digest, size_bytes)
	VALUES($1, $2, $3, $4)
	ON CONFLICT ON CONSTRAINT output_digest_key
	DO UPDATE SET id=output.id RETURNING id
	`
	var id int

	err := tx.QueryRow(stmt, a.Name, a.Type, a.Digest, a.SizeBytes).Scan(&id)
	if err != nil {
		return -1, errors.Wrapf(err, "db query %q failed", stmt)
	}

	return id, nil
}

func insertInputBuild(tx *sql.Tx, buildID, inputID int) error {
	const stmt = "INSERT into input_build VALUES($1, $2)"

	_, err := tx.Exec(stmt, buildID, inputID)

	return err
}

func insertInputIfNotExist(tx *sql.Tx, s *storage.Input) (int, error) {
	const stmt = `
	INSERT INTO input
	(url, digest)
	VALUES($1, $2)
	ON CONFLICT ON CONSTRAINT input_uniq
	DO UPDATE SET id=input.id RETURNING id
	`
	var id int

	err := tx.QueryRow(stmt, s.URL, s.Digest).Scan(&id)
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

func insertUpload(tx *sql.Tx, buildID, outputID int, url string, uploadDuration time.Duration) error {
	const stmt = `
	INSERT into upload
	(build_id, output_id, uri, upload_duration_msec)
	VALUES($1, $2, $3, $4)
	RETURNING id
	`

	_, err := tx.Exec(stmt, buildID, outputID, url, uploadDuration/time.Millisecond)
	return err
}

func saveOutput(tx *sql.Tx, buildID int, a *storage.Output) error {
	outputID, err := insertOutputIfNotExist(tx, a)
	if err != nil {
		return errors.Wrap(err, "storing output record failed")
	}

	err = insertUpload(tx, buildID, outputID, a.URI, a.UploadDuration)
	if err != nil {
		return errors.Wrap(err, "storing upload record failed")
	}

	return nil
}

// Save stores a build
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

	buildID, err := insertBuild(tx, appID, b)
	if err != nil {
		return errors.Wrap(err, "storing build record failed")
	}

	for _, a := range b.Outputs {
		if err := saveOutput(tx, buildID, a); err != nil {
			return err
		}
	}

	for _, s := range b.Inputs {
		inputID, err := insertInputIfNotExist(tx, s)
		if err != nil {
			return errors.Wrap(err, "storing input record failed")
		}

		err = insertInputBuild(tx, buildID, inputID)
		if err != nil {
			return errors.Wrap(err, "storing input_build failed")
		}
	}

	return nil
}

// FindLatestAppBuildByDigest returns the build id of a build for the
// application with the passed digest. If multiple builds exist, the one with
// the lastest stop_timestamp is returned.
// If no builds exist sql.ErrNoRows is returned
func (c *Client) FindLatestAppBuildByDigest(appName, totalInputDigest string) (int64, error) {
	const stmt = `
	 SELECT build.id
	 FROM application AS app
	 JOIN build AS build ON app.id = build.application_id
	 WHERE app.name = $1
	 ORDER BY build.stop_timestamp DESC LIMIT 1
	 `
	var id int64

	err := c.db.QueryRow(stmt, appName).Scan(&id)
	if err != nil {
		return -1, errors.Wrapf(err, "db query %q failed", stmt)
	}

	return id, nil
}

func (c *Client) populateOutputs(build *storage.Build, buildID int64) error {
	const stmt = `SELECT
			output.name, output.digest, output.type, output.size_bytes,
			upload.uri, upload.upload_duration_msec
		      FROM output
		      JOIN upload ON upload.output_id = output.id
		      WHERE upload.build_id = $1
		      `

	rows, err := c.db.Query(stmt, buildID)
	if err != nil {
		return errors.Wrapf(err, "db query %q failed", stmt)
	}

	for rows.Next() {
		var output storage.Output
		var uploadDurationMsec int64

		rows.Scan(
			&output.Name,
			&output.Digest,
			&output.Type,
			&output.SizeBytes,
			&output.URI,
			&uploadDurationMsec,
		)

		output.UploadDuration = time.Duration(uploadDurationMsec) * time.Millisecond
		build.Outputs = append(build.Outputs, &output)
	}

	if err := rows.Err(); err != nil {
		return errors.Wrap(err, "iterating over rows failed")
	}

	return nil
}

// GetBuildWithoutInputs retrieves a build from the database
func (c *Client) GetBuildWithoutInputs(id int64) (*storage.Build, error) {
	var build storage.Build

	const stmt = `
	 SELECT app.name,
		build.start_timestamp, build.stop_timestamp, build.total_input_digest
	 FROM application AS app
	 JOIN build ON app.id = build.application_id
	 WHERE build.id = $1`

	err := c.db.QueryRow(stmt, id).Scan(
		&build.AppName,
		&build.StartTimeStamp,
		&build.StopTimeStamp,
		&build.TotalInputDigest)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, storage.ErrNotExist
		}

		return nil, errors.Wrapf(err, "db query %q failed", stmt)
	}

	// TODO: handle the case that the build doesnt have any artifact and
	// this query fails
	if err := c.populateOutputs(&build, id); err != nil {
		return nil, errors.Wrap(err, "fetching build outputs failed")
	}

	return &build, err
}
