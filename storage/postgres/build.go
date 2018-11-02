package postgres

import (
	"database/sql"
	"fmt"

	"github.com/pkg/errors"

	"github.com/simplesurance/baur/storage"
)

const buildQueryWithoutInputsOutputs = `
SELECT application.id, application.name,
       build.id, build.start_timestamp, build.stop_timestamp, build.total_input_digest, 
       vcs.commit, vcs.dirty,
       (EXTRACT(EPOCH FROM (build.stop_timestamp - build.start_timestamp))::bigint * 1000000000) as duration
FROM application
JOIN build ON application.id = build.application_id
LEFT OUTER JOIN vcs ON vcs.id = build.vcs_id`

// GetBuildsWithoutInputs returns builds from the database
func (c *Client) GetBuildsWithoutInputs(filters []*storage.Filter, sorters []*storage.Sorter) (
	[]*storage.BuildWithDuration, error) {

	var builds []*storage.BuildWithDuration

	q := Query{
		BaseQuery: buildQueryWithoutInputsOutputs,
		Filters:   filters,
		Sorters:   sorters,
	}

	query, args, err := q.Compile()
	if err != nil {
		return nil, errors.Wrap(err, "compiling query string failed")
	}

	rows, err := c.Db.Query(query, args...)
	if err != nil {
		return nil, errors.Wrapf(err, "db query '%s' (%q) failed", query, args)
	}

	for rows.Next() {
		build, err := scanBuildRows(rows)
		if err != nil {
			rows.Close()
			return nil, errors.Wrapf(err, "scanning result of db query '%s' (%q) failed", query, args)
		}

		builds = append(builds, build)
	}

	if err := rows.Err(); err != nil {
		return nil, errors.Wrap(err, "iterating over db results failed:")
	}

	return builds, nil
}

func scanBuildRows(rows *sql.Rows) (*storage.BuildWithDuration, error) {
	var build storage.BuildWithDuration

	err := rows.Scan(
		&build.Build.Application.ID,
		&build.Build.Application.Name,
		&build.Build.ID,
		&build.Build.StartTimeStamp,
		&build.Build.StopTimeStamp,
		&build.Build.TotalInputDigest,
		&build.Build.VCSState.CommitID,
		&build.Build.VCSState.IsDirty,
		&build.Duration,
	)
	if err != nil {
		return nil, err
	}

	return &build, nil
}

// GetLatestBuildByDigest returns the build id of a build for the application
// with the passed digest. If multiple builds exist, the one with the lastest
// stop_timestamp is returned.
// Inputs are not fetched from the database.
// If no builds exist storage.ErrNotExist is returned
func (c *Client) GetLatestBuildByDigest(appName, totalInputDigest string) (*storage.BuildWithDuration, error) {
	const query = buildQueryWithoutInputsOutputs + `
	 WHERE application.name = $1 AND build.total_input_digest = $2
	 ORDER BY build.stop_timestamp DESC LIMIT 1
	 `

	rows, err := c.Db.Query(query, appName, totalInputDigest)
	if err != nil {
		return nil, errors.Wrapf(err, "db query '%s' failed", query)
	}

	if !rows.Next() {
		return nil, storage.ErrNotExist
	}

	if err := rows.Err(); err != nil {
		return nil, errors.Wrap(err, "iterating over db results failed:")
	}

	build, err := scanBuildRows(rows)
	rows.Close()
	if err != nil {
		return nil, errors.Wrapf(err, "scanning result of db query '%s' failed", query)
	}

	builds, err := c.GetBuildOutputs(build.ID)
	if err != nil {
		return nil, errors.Wrap(err, "fetching build outputs failed")
	}

	build.Outputs = builds

	return build, err
}

// GetBuildWithoutInputs retrieves a single build from the database
func (c *Client) GetBuildWithoutInputs(id int) (*storage.BuildWithDuration, error) {
	builds, err := c.GetBuildsWithoutInputs([]*storage.Filter{
		&storage.Filter{
			Field:    storage.FieldBuildID,
			Operator: storage.OpEQ,
			Value:    id,
		}}, nil)
	if err != nil {
		return nil, err
	}

	if len(builds) == 0 {
		return nil, storage.ErrNotExist
	}

	if len(builds) > 1 {
		panic(fmt.Sprintf("GetBuilds returned >%d records for build id '%d', expected max. 1", len(builds), id))
	}

	return builds[0], nil
}
