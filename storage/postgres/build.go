package postgres

import (
	"database/sql"

	"github.com/pkg/errors"

	"github.com/simplesurance/baur/storage"
)

// GetBuilds returns builds from the database
func (c *Client) GetBuilds(filters []*storage.Filter, sorters []*storage.Sorter) (
	[]*storage.BuildWithDuration, error) {
	const baseQuery = `
		SELECT application.id, application.name,
		       build.id, build.start_timestamp, build.total_input_digest, 
		       vcs.commit, vcs.dirty,
		       (EXTRACT(EPOCH FROM (build.stop_timestamp - build.start_timestamp))::bigint * 1000000000) as duration
		FROM application
		JOIN build ON application.id = build.application_id
		LEFT OUTER JOIN vcs ON vcs.id = build.vcs_id`

	var builds []*storage.BuildWithDuration

	q := Query{
		BaseQuery: baseQuery,
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
		build, err := scanBuildRow(rows)
		if err != nil {
			return nil, errors.Wrapf(err, "scanning result of db query '%s' (%q) failed", query, args)
		}

		builds = append(builds, build)
	}

	return builds, nil
}

func scanBuildRow(rows *sql.Rows) (*storage.BuildWithDuration, error) {
	var (
		build    storage.BuildWithDuration
		commitID sql.NullString
		isDirty  sql.NullBool
	)

	err := rows.Scan(
		&build.Build.Application.ID,
		&build.Build.Application.Name,
		&build.Build.ID,
		&build.Build.StartTimeStamp,
		&build.Build.TotalInputDigest,
		&commitID,
		&isDirty,
		&build.Duration,
	)
	if err != nil {
		return nil, err
	}

	if commitID.Valid {
		build.Build.VCSState.CommitID = commitID.String
		build.Build.VCSState.IsDirty = isDirty.Bool
	}

	return &build, nil
}
