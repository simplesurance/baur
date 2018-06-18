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

func insertArtifact(tx *sql.Tx, buildID int, a *storage.Artifact) (int, error) {
	const stmt = `
	INSERT INTO artifact
	(build_id, name, type, url, hash, size_bytes, upload_duration_msec)
	VALUES($1, $2, $3, $4, $5, $6, $7)
	RETURNING id;
	`

	var id int

	r := tx.QueryRow(stmt, buildID, a.Name, a.Type, a.URL, a.Hash, a.SizeBytes, a.UploadDuration/time.Millisecond)
	if err := r.Scan(&id); err != nil {
		return -1, err
	}

	return id, nil
}

// TODO can we use the lastinsertedId function from the result that tx.Exec()
// returns instead of using QueryRow()??
func insertSourceBuild(tx *sql.Tx, buildID, sourceID int) error {
	const stmt = "INSERT into source_build VALUES($1, $2)"

	_, err := tx.Exec(stmt, buildID, sourceID)

	return err
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

	var id int

	_, err := tx.Exec("SAVEPOINT ssp")
	if err != nil {
		return -1, errors.Wrap(err, "creating savepoint failed")
	}

	r := tx.QueryRow(insertStmt, s.RelativePath, s.Hash)
	insertErr := r.Scan(&id)
	if insertErr == nil {
		return id, nil
	}

	// row already exist, TODO: only rollback and continue if it's
	// an already exist error
	_, err = tx.Exec("ROLLBACK TO ssp")
	if err != nil {
		return -1, errors.Wrapf(err, "rolling back transaction after insert error %q failed", insertErr)
	}

	r = tx.QueryRow(selectStmt, s.RelativePath, s.Hash)
	if err := r.Scan(&id); err != nil {
		return -1, errors.Wrapf(err, "selecting source record failed after insert failed: %s", insertErr)
	}

	return id, nil

}

func insertAppIfNotExist(tx *sql.Tx, appName string) (int, error) {
	const insertStmt = `
	INSERT INTO application
	(application_name)
	VALUES($1)
	RETURNING id;
	`
	const selectStmt = "SELECT id FROM application WHERE application_name = $1;"

	var id int

	_, err := tx.Exec("SAVEPOINT asp;")
	if err != nil {
		return -1, errors.Wrap(err, "creating savepoint failed")
	}

	r := tx.QueryRow(insertStmt, appName)
	insertErr := r.Scan(&id)
	if insertErr == nil {
		return id, nil
	}

	_, err = tx.Exec("ROLLBACK TO asp")
	if err != nil {
		return -1, errors.Wrapf(err, "rolling back transaction after insert error %q failed", insertErr)
	}

	r = tx.QueryRow(selectStmt, appName)
	if err := r.Scan(&id); err != nil {
		return -1, errors.Wrap(err, "selecting application record failed")
	}

	return id, nil
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
		return errors.Wrap(err, "storing build failed")
	}

	for _, a := range b.Artifacts {
		_, err := insertArtifact(tx, buildID, a)
		if err != nil {
			return errors.Wrap(err, "storing artifact failed")
		}
	}

	for _, s := range b.Sources {
		sourceID, err := insertSourceIfNotExist(tx, s)
		if err != nil {
			return errors.Wrap(err, "storing source failed")
		}

		err = insertSourceBuild(tx, buildID, sourceID)
		if err != nil {
			return errors.Wrap(err, "storing source_build failed")
		}
	}

	return nil
}
