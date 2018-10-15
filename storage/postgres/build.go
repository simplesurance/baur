package postgres

import (
	"database/sql"
	"fmt"

	"github.com/pkg/errors"
	"github.com/simplesurance/baur/storage"
)

// BuildLsFieldsMap is the map of fields => strings
var BuildLsFieldsMap = SQLFields{
	storage.FieldBuildID:               "build.id",
	storage.FieldDuration:              "duration",
	storage.FieldApplicationName:       "application.name",
	storage.FieldBuildStartDatetime:    "build.start_timestamp",
	storage.FieldBuildTotalInputDigest: "build.total_input_digest",
	storage.FieldOne:                   "1",
}

// BuildLsFilterOperators is the map of operators => strings
var BuildLsFilterOperators = SQLFilterOperators{
	storage.OperatorEq:  "=",
	storage.OperatorGt:  ">",
	storage.OperatorLt:  "<",
	storage.OperatorGte: ">=",
	storage.OperatorLte: "<=",
}

// BuildLsSQLMap is an actual implementation of SQLStringer
var BuildLsSQLMap = SQLMap{
	Fields:    BuildLsFieldsMap,
	Operators: BuildLsFilterOperators,
}

// GetBuilds fetches builds
func (c *Client) GetBuilds(filters []storage.CanFilter, sorters []storage.CanSort) (
	[]*storage.BuildWithDuration, error) {
	queryStr := fmt.Sprintf(`
		SELECT
			application.id, application.name,
			build.id, build.start_timestamp, build.total_input_digest, 
			vcs.commit, vcs.dirty,
			EXTRACT(EPOCH FROM (build.stop_timestamp - build.start_timestamp)) as duration
		FROM application
		JOIN build ON application.id = build.application_id
		LEFT OUTER JOIN vcs ON vcs.id = build.vcs_id
		WHERE %s
		ORDER BY %s`,
		WrapKey(PlaceholderFilters),
		WrapKey(PlaceholderSorters),
	)

	q := NewQuery(queryStr, BuildLsSQLMap)

	// todo find a way to strip WHERE and ORDER BY clauses from the compiled query when no filters / sorters
	// so that we don't have to rely on the following hack. This is like "WHERE 1 = 1 AND ...":
	defaultFilter := NewFilter(storage.FieldOne, storage.OperatorEq, "1")
	// prepend the default 1=1 filter to the incoming filters list
	filters = append([]storage.CanFilter{defaultFilter}, filters...)

	err := q.SetFilters(filters)
	if err != nil {
		return nil, errors.Wrap(err, "problem with filters")
	}

	if len(sorters) == 0 {
		sorter := NewSorter(storage.FieldBuildStartDatetime, storage.OrderDesc)
		sorters = []storage.CanSort{sorter}
	}
	q.SetSorters(sorters)

	rowValues, err := RunSelectQuery(*c, *q, rowParser)
	if err != nil {
		return nil, errors.Wrap(err, "error retrieving builds")
	}

	buildsWithDuration, err := getBuildsFromQueryResults(rowValues)
	if err != nil {
		return nil, errors.Wrap(err, "error converting at least one of the query results to build")
	}

	return buildsWithDuration, nil
}

func rowParser(rows *sql.Rows) (interface{}, error) {
	var (
		buildWithDuration storage.BuildWithDuration
		commitID          sql.NullString
		isDirty           sql.NullBool
	)

	err := rows.Scan(
		&buildWithDuration.Build.Application.ID,
		&buildWithDuration.Build.Application.Name,
		&buildWithDuration.Build.ID,
		&buildWithDuration.Build.StartTimeStamp,
		&buildWithDuration.Build.TotalInputDigest,
		&commitID,
		&isDirty,
		&buildWithDuration.Duration,
	)
	if err != nil {
		return nil, err
	}

	if commitID.Valid {
		buildWithDuration.Build.VCSState.CommitID = commitID.String
		buildWithDuration.Build.VCSState.IsDirty = isDirty.Bool
	}

	return buildWithDuration, nil
}

func getBuildsFromQueryResults(rowValues []interface{}) (buildsWithDuration []*storage.BuildWithDuration, err error) {
	for _, value := range rowValues {
		buildWithDuration, ok := value.(storage.BuildWithDuration)
		if !ok {
			return nil, fmt.Errorf("strange build retrieved from db: %q", value)
		}
		buildsWithDuration = append(buildsWithDuration, &buildWithDuration)
	}

	return
}
