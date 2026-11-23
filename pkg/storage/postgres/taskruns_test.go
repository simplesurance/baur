//go:build dbtest

package postgres

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/simplesurance/baur/v5/pkg/storage"
)

func taskRunIDs(t *testing.T, clt *Client, filters []*storage.Filter, sorters []*storage.Sorter, limit uint) []int {
	t.Helper()

	var ids []int

	err := clt.TaskRuns(ctx, filters, sorters, limit, func(tr *storage.TaskRunWithID) error {
		ids = append(ids, tr.ID)
		return nil
	})
	require.NoError(t, err)

	return ids
}

func TestTaskRunsFiltersSortersAndLimit(t *testing.T) {
	clt, cleanup := newTestClient(t)
	defer cleanup()
	require.NoError(t, clt.Init(ctx))

	base := time.Now().Truncate(time.Second).Add(-time.Hour)

	// start times ascending: r1 < r2 < r3 < r4
	r1 := saveTestRun(t, clt, "a", "build", "d1", base)
	r2 := saveTestRun(t, clt, "a", "test", "d2", base.Add(time.Minute))
	r3 := saveTestRun(t, clt, "b", "build", "d3", base.Add(2*time.Minute))
	r4 := saveTestRun(t, clt, "b", "build", "d4", base.Add(3*time.Minute))

	eq := func(f storage.Field, v any) *storage.Filter {
		return &storage.Filter{Field: f, Operator: storage.OpEQ, Value: v}
	}
	sortBy := func(f storage.Field, o storage.Order) []*storage.Sorter {
		return []*storage.Sorter{{Field: f, Order: o}}
	}

	t.Run("sort by start time ascending and descending", func(t *testing.T) {
		assert.Equal(t, []int{r1, r2, r3, r4},
			taskRunIDs(t, clt, nil, sortBy(storage.FieldStartTime, storage.OrderAsc), storage.NoLimit))
		assert.Equal(t, []int{r4, r3, r2, r1},
			taskRunIDs(t, clt, nil, sortBy(storage.FieldStartTime, storage.OrderDesc), storage.NoLimit))
	})

	t.Run("multiple sorters, later ones break ties", func(t *testing.T) {
		sorters := []*storage.Sorter{
			{Field: storage.FieldApplicationName, Order: storage.OrderDesc},
			{Field: storage.FieldStartTime, Order: storage.OrderAsc},
		}

		assert.Equal(t, []int{r3, r4, r1, r2}, taskRunIDs(t, clt, nil, sorters, storage.NoLimit))
	})

	t.Run("limit restricts the number of results", func(t *testing.T) {
		sorters := sortBy(storage.FieldStartTime, storage.OrderDesc)

		assert.Equal(t, []int{r4, r3}, taskRunIDs(t, clt, nil, sorters, 2))
		assert.Equal(t, []int{r4}, taskRunIDs(t, clt, nil, sorters, 1))
		assert.Len(t, taskRunIDs(t, clt, nil, sorters, 100), 4)
	})

	t.Run("start time greater than and less than", func(t *testing.T) {
		sorters := sortBy(storage.FieldStartTime, storage.OrderAsc)

		gt := []*storage.Filter{{Field: storage.FieldStartTime, Operator: storage.OpGT, Value: base.Add(time.Minute)}}
		assert.Equal(t, []int{r3, r4}, taskRunIDs(t, clt, gt, sorters, storage.NoLimit))

		lt := []*storage.Filter{{Field: storage.FieldStartTime, Operator: storage.OpLT, Value: base.Add(time.Minute)}}
		assert.Equal(t, []int{r1}, taskRunIDs(t, clt, lt, sorters, storage.NoLimit))
	})

	t.Run("filters are combined with AND", func(t *testing.T) {
		f := []*storage.Filter{
			eq(storage.FieldApplicationName, "b"),
			{Field: storage.FieldStartTime, Operator: storage.OpGT, Value: base.Add(2 * time.Minute)},
		}

		assert.Equal(t, []int{r4}, taskRunIDs(t, clt, f, nil, storage.NoLimit))
	})

	t.Run("filter by task run id", func(t *testing.T) {
		assert.Equal(t, []int{r2}, taskRunIDs(t, clt, []*storage.Filter{eq(storage.FieldID, r2)}, nil, storage.NoLimit))
	})

	t.Run("no match returns ErrNotExist", func(t *testing.T) {
		err := clt.TaskRuns(ctx,
			[]*storage.Filter{eq(storage.FieldApplicationName, "does-not-exist")},
			nil, storage.NoLimit,
			func(*storage.TaskRunWithID) error { return nil },
		)
		require.ErrorIs(t, err, storage.ErrNotExist)
	})

	t.Run("callback error stops the iteration and is returned", func(t *testing.T) {
		calls := 0
		err := clt.TaskRuns(ctx, nil, sortBy(storage.FieldID, storage.OrderAsc), storage.NoLimit,
			func(*storage.TaskRunWithID) error {
				calls++
				return assert.AnError
			},
		)

		require.ErrorIs(t, err, assert.AnError)
		assert.Equal(t, 1, calls)
	})

	t.Run("invalid filter field returns an error", func(t *testing.T) {
		err := clt.TaskRuns(ctx,
			[]*storage.Filter{{Field: storage.FieldUndefined, Operator: storage.OpEQ, Value: 1}},
			nil, storage.NoLimit,
			func(*storage.TaskRunWithID) error { return nil },
		)
		require.Error(t, err)
		assert.NotErrorIs(t, err, storage.ErrNotExist)
	})
}

func TestTaskRunsInputFilters(t *testing.T) {
	clt, cleanup := newTestClient(t)
	defer cleanup()
	require.NoError(t, clt.Init(ctx))

	now := time.Now().Truncate(time.Second)

	withFile := saveTestRun(t, clt, "a", "build", "d1", now, withInputs(storage.Inputs{
		Files: []*storage.InputFile{{Path: "main.go", Digest: "1"}, {Path: "util.go", Digest: "2"}},
	}))
	withStr := saveTestRun(t, clt, "a", "build", "d2", now.Add(time.Minute), withInputs(storage.Inputs{
		Strings: []*storage.InputString{{String: "v1.2.3", Digest: "3"}},
	}))
	_ = saveTestRun(t, clt, "a", "build", "d3", now.Add(2*time.Minute), withInputs(storage.Inputs{
		Files: []*storage.InputFile{{Path: "other.go", Digest: "4"}},
	}))

	t.Run("input file path", func(t *testing.T) {
		f := []*storage.Filter{{Field: storage.FieldInputFilePath, Operator: storage.OpEQ, Value: "main.go"}}

		assert.Equal(t, []int{withFile}, taskRunIDs(t, clt, f, nil, storage.NoLimit))
	})

	t.Run("input string", func(t *testing.T) {
		f := []*storage.Filter{{Field: storage.FieldInputString, Operator: storage.OpEQ, Value: "v1.2.3"}}

		assert.Equal(t, []int{withStr}, taskRunIDs(t, clt, f, nil, storage.NoLimit))
	})

	t.Run("a run with multiple matching files is returned once per file", func(t *testing.T) {
		f := []*storage.Filter{{
			Field: storage.FieldInputFilePath, Operator: storage.OpIN,
			Value: []string{"main.go", "util.go"},
		}}

		assert.Equal(t, []int{withFile, withFile}, taskRunIDs(t, clt, f, nil, storage.NoLimit))
	})

	t.Run("file and string filters cannot be combined", func(t *testing.T) {
		err := clt.TaskRuns(ctx,
			[]*storage.Filter{
				{Field: storage.FieldInputFilePath, Operator: storage.OpEQ, Value: "main.go"},
				{Field: storage.FieldInputString, Operator: storage.OpEQ, Value: "v1.2.3"},
			},
			nil, storage.NoLimit,
			func(*storage.TaskRunWithID) error { return nil },
		)

		require.Error(t, err)
		assert.NotErrorIs(t, err, storage.ErrNotExist)
	})

	t.Run("no run has the input", func(t *testing.T) {
		err := clt.TaskRuns(ctx,
			[]*storage.Filter{{Field: storage.FieldInputFilePath, Operator: storage.OpEQ, Value: "nope.go"}},
			nil, storage.NoLimit,
			func(*storage.TaskRunWithID) error { return nil },
		)

		require.ErrorIs(t, err, storage.ErrNotExist)
	})
}

func TestInputsRoundtripAllInputTypes(t *testing.T) {
	clt, cleanup := newTestClient(t)
	defer cleanup()
	require.NoError(t, clt.Init(ctx))

	in := storage.Inputs{
		Files:                []*storage.InputFile{{Path: "a.go", Digest: "f1"}, {Path: "b.go", Digest: "f2"}},
		Strings:              []*storage.InputString{{String: "s", Digest: "s1"}},
		EnvironmentVariables: []*storage.InputEnvVar{{Name: "FOO", Digest: "e1"}, {Name: "BAR", Digest: "e2"}},
		TaskInfo:             []*storage.InputTaskInfo{{Name: "app.other", Digest: "t1"}},
	}

	id := saveTestRun(t, clt, "a", "build", "d1", time.Now(), withInputs(in))

	got, err := clt.Inputs(ctx, id)
	require.NoError(t, err)

	assert.ElementsMatch(t, in.Files, got.Files)
	assert.ElementsMatch(t, in.Strings, got.Strings)
	assert.ElementsMatch(t, in.EnvironmentVariables, got.EnvironmentVariables)
	assert.ElementsMatch(t, in.TaskInfo, got.TaskInfo)
}

func TestSaveTaskRunSharesRecordsOfIdenticalInputsAndOutputs(t *testing.T) {
	clt, cleanup := newTestClient(t)
	defer cleanup()
	require.NoError(t, clt.Init(ctx))

	in := storage.Inputs{
		Files:                []*storage.InputFile{{Path: "a.go", Digest: "f1"}},
		Strings:              []*storage.InputString{{String: "s", Digest: "s1"}},
		EnvironmentVariables: []*storage.InputEnvVar{{Name: "FOO", Digest: "e1"}},
		TaskInfo:             []*storage.InputTaskInfo{{Name: "app.other", Digest: "t1"}},
	}
	out := func() *storage.Output {
		return fileOutput("bin", "o1", upload(storage.UploadMethodS3, "s3://b/k"))
	}

	now := time.Now()
	saveTestRun(t, clt, "a", "build", "d1", now, withInputs(in), withOutputs(out()), withRevision("r", true))
	saveTestRun(t, clt, "a", "build", "d1", now.Add(time.Minute), withInputs(in), withOutputs(out()), withRevision("r", true))

	count := func(table string) int {
		var n int
		require.NoError(t, clt.db.QueryRow(ctx, "SELECT COUNT(*) FROM "+table).Scan(&n))

		return n
	}

	assert.Equal(t, 2, count("task_run"))

	for _, table := range []string{
		"application", "task", "vcs", "input_file", "input_string", "input_env_var", "input_task", "output",
	} {
		assert.Equalf(t, 1, count(table), "table %s must have a single shared row", table)
	}
}

func TestSaveTaskRunDistinguishesVCSDirtyState(t *testing.T) {
	clt, cleanup := newTestClient(t)
	defer cleanup()
	require.NoError(t, clt.Init(ctx))

	now := time.Now()
	clean := saveTestRun(t, clt, "a", "build", "d1", now, withRevision("r", false))
	dirty := saveTestRun(t, clt, "a", "build", "d2", now.Add(time.Minute), withRevision("r", true))

	got, err := clt.TaskRun(ctx, clean)
	require.NoError(t, err)
	assert.False(t, got.VCSIsDirty)
	assert.Equal(t, "r", got.VCSRevision)

	got, err = clt.TaskRun(ctx, dirty)
	require.NoError(t, err)
	assert.True(t, got.VCSIsDirty)
}

func TestLatestTaskRunByDigestSelection(t *testing.T) {
	clt, cleanup := newTestClient(t)
	defer cleanup()
	require.NoError(t, clt.Init(ctx))

	now := time.Now().Truncate(time.Second)

	older := saveTestRun(t, clt, "a", "build", "same", now)
	newer := saveTestRun(t, clt, "a", "build", "same", now.Add(time.Minute))
	saveTestRun(t, clt, "a", "build", "other-digest", now.Add(time.Hour))
	saveTestRun(t, clt, "a", "test", "same", now.Add(time.Hour))
	saveTestRun(t, clt, "b", "build", "same", now.Add(time.Hour))

	t.Run("newest run for app, task and digest is returned", func(t *testing.T) {
		got, err := clt.LatestTaskRunByDigest(ctx, "a", "build", "same")
		require.NoError(t, err)

		assert.Equal(t, newer, got.ID)
		assert.NotEqual(t, older, got.ID)
		assert.Equal(t, "a", got.ApplicationName)
		assert.Equal(t, "build", got.TaskName)
		assert.Equal(t, "same", got.TotalInputDigest)
	})

	t.Run("app, task and digest must all match", func(t *testing.T) {
		for _, tc := range [][3]string{
			{"x", "build", "same"},
			{"a", "x", "same"},
			{"a", "build", "x"},
		} {
			_, err := clt.LatestTaskRunByDigest(ctx, tc[0], tc[1], tc[2])
			require.ErrorIsf(t, err, storage.ErrNotExist, "%v", tc)
		}
	})

	// The query does not filter on the result. Callers must only store
	// runs that must count as "task already ran".
	t.Run("failed runs are returned as well", func(t *testing.T) {
		failed := saveTestRun(t, clt, "f", "build", "d", now, withResult(storage.ResultFailure))

		got, err := clt.LatestTaskRunByDigest(ctx, "f", "build", "d")
		require.NoError(t, err)
		assert.Equal(t, failed, got.ID)
		assert.Equal(t, storage.ResultFailure, got.Result)
	})
}

// A run that only has TaskInfo inputs has inputs.
func TestInputsOfRunWithOnlyTaskInfoReturnsTheTaskInfo(t *testing.T) {
	clt, cleanup := newTestClient(t)
	defer cleanup()
	require.NoError(t, clt.Init(ctx))

	id := saveTestRun(t, clt, "a", "build", "d1", time.Now(), withInputs(storage.Inputs{
		TaskInfo: []*storage.InputTaskInfo{{Name: "app.other", Digest: "t1"}},
	}))

	got, err := clt.Inputs(ctx, id)

	require.NoError(t, err)
	require.Len(t, got.TaskInfo, 1)
	assert.Equal(t, "app.other", got.TaskInfo[0].Name)
	assert.Equal(t, "t1", got.TaskInfo[0].Digest)
}
