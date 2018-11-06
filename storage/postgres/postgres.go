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
	Db *sql.DB
}

// New establishes a connection a postgres db
func New(url string) (*Client, error) {
	db, err := sql.Open("postgres", url)
	if err != nil {
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

// GetBuildOutputs returns build outputs
func (c *Client) GetBuildOutputs(buildID int) ([]*storage.Output, error) {
	const stmt = `SELECT
			output.name, output.digest, output.type, output.size_bytes,
			upload.id, upload.uri, upload.upload_duration_ns
		      FROM output
		      JOIN build_output ON output.id = build_output.output_id
		      JOIN upload ON upload.build_output_id = build_output.id
		      WHERE build_output.build_id = $1
		      `

	rows, err := c.Db.Query(stmt, buildID)
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
			&output.Upload.URI,
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

// GetBuild returns what the name says
func (c *Client) GetBuild(id int) (build *storage.BuildWithDuration, err error) {
	build = &storage.BuildWithDuration{}
	row := c.Db.QueryRow(`SELECT build.id, build.start_timestamp, build.stop_timestamp, 
       build.total_input_digest, vcs.commit, vcs.dirty,
       (EXTRACT(EPOCH FROM (build.stop_timestamp - build.start_timestamp))::bigint * 1000000000) as duration
       FROM build LEFT OUTER JOIN vcs ON vcs.id = build.vcs_id
       WHERE build.id = $1`, id)
	if err = row.Scan(
		&build.ID,
		&build.StartTimeStamp,
		&build.StopTimeStamp,
		&build.TotalInputDigest,
		&build.VCSState.CommitID,
		&build.VCSState.IsDirty,
		&build.Duration,
	); err != nil {
		return nil, errors.Wrap(err, "query error")
	}

	return
}
