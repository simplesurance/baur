package postgres

import (
	"database/sql"
	"fmt"
	"time"

	_ "github.com/lib/pq" // postgresql
	"github.com/pkg/errors"
	"github.com/rs/xid"

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
	(application_id, start_timestamp, stop_timestamp, total_src_hash)
	VALUES($1, $2, $3, $4)
	RETURNING id;`

	var id int

	r := tx.QueryRow(stmt, appID, b.StartTimeStamp, b.StopTimeStamp, b.TotalSrcHash)

	if err := r.Scan(&id); err != nil {
		return -1, err
	}

	return id, nil
}

func insertArtifactIfNotExist(tx *sql.Tx, a *storage.Artifact) (int, error) {
	const insertStmt = `
	INSERT INTO artifact
	(name, type, hash, size_bytes)
	VALUES($1, $2, $3, $4)
	RETURNING id;
	`

	const selectStmt = `
	SELECT id FROM artifact 
	WHERE name = $1 AND hash = $2 AND size_bytes = $3;
	`

	return insertIfNotExist(tx,
		insertStmt, []interface{}{a.Name, a.Type, a.Hash, a.SizeBytes},
		selectStmt, []interface{}{a.Name, a.Hash, a.SizeBytes})
}

func insertSourceBuild(tx *sql.Tx, buildID, sourceID int) error {
	const stmt = "INSERT into source_build VALUES($1, $2)"

	_, err := tx.Exec(stmt, buildID, sourceID)

	return err
}

func insertIfNotExist(
	tx *sql.Tx,
	insertStmt string,
	insertArgs []interface{},
	selectStmt string,
	selectArgs []interface{},
) (int, error) {
	var id int
	savepointName := xid.New().String()

	_, err := tx.Exec(fmt.Sprintf("SAVEPOINT %s", savepointName))
	if err != nil {
		return -1, errors.Wrapf(err, "creating savepoint %q failed", savepointName)
	}

	r := tx.QueryRow(insertStmt, insertArgs...)
	insertErr := r.Scan(&id)
	if insertErr == nil {
		return id, nil
	}

	// row already exist, TODO: only rollback and continue if it's
	// an already exist error
	_, err = tx.Exec(fmt.Sprintf("ROLLBACK TO %s", savepointName))
	if err != nil {
		return -1, errors.Wrapf(err, "rolling back transaction after insert error %q failed", insertErr)
	}

	r = tx.QueryRow(selectStmt, selectArgs...)
	if err := r.Scan(&id); err != nil {
		return -1, errors.Wrapf(err, "selecting source record failed after insert failed: %s", insertErr)
	}

	return id, nil
}

func insertSourceIfNotExist(tx *sql.Tx, s *storage.Source) (int, error) {
	const insertStmt = `
	INSERT INTO source
	(relative_path, hash)
	VALUES($1, $2)
	RETURNING id;
	`

	const selectStmt = `
	SELECT id FROM source
	WHERE relative_path = $1 AND hash = $2;
	`

	return insertIfNotExist(tx,
		insertStmt, []interface{}{s.RelativePath, s.Hash},
		selectStmt, []interface{}{s.RelativePath, s.Hash})
}

func insertAppIfNotExist(tx *sql.Tx, appName string) (int, error) {
	const insertStmt = `
	INSERT INTO application
	(application_name)
	VALUES($1)
	RETURNING id;
	`
	const selectStmt = "SELECT id FROM application WHERE application_name = $1;"

	return insertIfNotExist(tx,
		insertStmt, []interface{}{appName},
		selectStmt, []interface{}{appName})
}

func insertArtifactBuild(tx *sql.Tx, buildID, artifactID int) error {
	const stmt = "INSERT into artifact_build VALUES($1, $2)"

	_, err := tx.Exec(stmt, buildID, artifactID)

	return err
}

func insertUpload(tx *sql.Tx, artifactID int, url string, uploadDuration time.Duration) error {
	const stmt = `
	INSERT into upload
	(artifact_id, uri, upload_duration_msec)
	VALUES($1, $2, $3)
	RETURNING id
	`

	_, err := tx.Exec(stmt, artifactID, url, uploadDuration/time.Millisecond)
	return err
}

func saveArtifact(tx *sql.Tx, buildID int, a *storage.Artifact) error {
	artifactID, err := insertArtifactIfNotExist(tx, a)
	if err != nil {
		return errors.Wrap(err, "storing artifact record failed")
	}

	err = insertArtifactBuild(tx, buildID, artifactID)
	if err != nil {
		return errors.Wrap(err, "storing artifact_build record failed")
	}

	err = insertUpload(tx, artifactID, a.URI, a.UploadDuration)
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

	for _, a := range b.Artifacts {
		if err := saveArtifact(tx, buildID, a); err != nil {
			return err
		}
	}

	for _, s := range b.Sources {
		sourceID, err := insertSourceIfNotExist(tx, s)
		if err != nil {
			return errors.Wrap(err, "storing source record failed")
		}

		err = insertSourceBuild(tx, buildID, sourceID)
		if err != nil {
			return errors.Wrap(err, "storing source_build failed")
		}
	}

	return nil
}
