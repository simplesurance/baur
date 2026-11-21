package baur

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/simplesurance/baur/v5/pkg/cfg"
	"github.com/simplesurance/baur/v5/pkg/storage"
)

func TestStatusEvaluatorFailsWhenTaskInfoAndLookupStrIsUsed(t *testing.T) {
	te := NewTaskStatusEvaluator("", nil, nil, "blubb")

	task := Task{
		UnresolvedInputs: &cfg.Input{
			TaskInfos: []cfg.TaskInfo{{TaskName: "abc"}},
		},
	}

	_, _, _, err := te.Status(t.Context(), &task)
	require.ErrorContains(t, err, "'--lookup-input-str' is unsupported")
}

func TestReplaceInputStrings(t *testing.T) {
	inputs := []Input{
		NewInputString("before"),
	}
	result := replaceInputStrings(NewInputs(inputs), []Input{NewInputString("after")})
	require.Len(t, result.inputs, 1)

	require.Contains(t, result.inputs[0].String(), "after")
}

type fakeRunLookupStorer struct {
	storage.Storer

	// lookup is called for every LatestTaskRunByDigest call, n is the 0-based call number.
	lookup  func(n int, appName, taskName, digest string) (*storage.TaskRunWithID, error)
	calls   int
	digests []string
}

func (f *fakeRunLookupStorer) LatestTaskRunByDigest(
	_ context.Context,
	appName, taskName, totalInputDigest string,
) (*storage.TaskRunWithID, error) {
	n := f.calls
	f.calls++
	f.digests = append(f.digests, totalInputDigest)

	return f.lookup(n, appName, taskName, totalInputDigest)
}

func newStatusTestTask() *Task {
	return &Task{
		ID:               "app.build",
		AppName:          "app",
		Name:             "build",
		UnresolvedInputs: &cfg.Input{},
	}
}

func newStatusTestEvaluator(st storage.Storer, lookupStr string) *TaskStatusEvaluator {
	resolver := NewInputResolver(&DummyGitUntrackedFilesResolver{}, "", []Input{NewInputString("orig")}, false)

	return NewTaskStatusEvaluator("", st, resolver, lookupStr)
}

func inputStrings(in *Inputs) []string {
	res := make([]string, 0, len(in.Inputs()))
	for _, i := range in.Inputs() {
		res = append(res, i.String())
	}

	return res
}

func TestStatusEvaluatorStatus(t *testing.T) {
	ctx := context.Background()
	existing := &storage.TaskRunWithID{ID: 7}

	t.Run("run with the same input digest exists", func(t *testing.T) {
		st := &fakeRunLookupStorer{lookup: func(int, string, string, string) (*storage.TaskRunWithID, error) {
			return existing, nil
		}}

		status, inputs, run, err := newStatusTestEvaluator(st, "").Status(ctx, newStatusTestTask())
		require.NoError(t, err)

		assert.Equal(t, TaskStatusRunExist, status)
		assert.Same(t, existing, run)
		assert.Contains(t, inputStrings(inputs), "string:orig")

		wantDigest, err := inputs.Digest()
		require.NoError(t, err)
		assert.Equal(t, []string{wantDigest.String()}, st.digests, "storage is queried with the total input digest")
	})

	t.Run("the storage is queried for the app and task of the task", func(t *testing.T) {
		var gotApp, gotTask string
		st := &fakeRunLookupStorer{lookup: func(_ int, app, task, _ string) (*storage.TaskRunWithID, error) {
			gotApp, gotTask = app, task
			return nil, storage.ErrNotExist
		}}

		_, _, _, err := newStatusTestEvaluator(st, "").Status(ctx, newStatusTestTask())
		require.NoError(t, err)

		assert.Equal(t, "app", gotApp)
		assert.Equal(t, "build", gotTask)
	})

	t.Run("no run exists means execution is pending", func(t *testing.T) {
		st := &fakeRunLookupStorer{lookup: func(int, string, string, string) (*storage.TaskRunWithID, error) {
			return nil, storage.ErrNotExist
		}}

		status, inputs, run, err := newStatusTestEvaluator(st, "").Status(ctx, newStatusTestTask())
		require.NoError(t, err)

		assert.Equal(t, TaskStatusExecutionPending, status)
		assert.Nil(t, run)
		assert.NotNil(t, inputs)
		assert.Equal(t, 1, st.calls)
	})

	t.Run("a storage error is returned and is not treated as pending", func(t *testing.T) {
		boom := errors.New("db down")
		st := &fakeRunLookupStorer{lookup: func(int, string, string, string) (*storage.TaskRunWithID, error) {
			return nil, boom
		}}

		status, inputs, run, err := newStatusTestEvaluator(st, "").Status(ctx, newStatusTestTask())

		require.ErrorIs(t, err, boom)
		assert.Equal(t, TaskStatusUndefined, status)
		assert.Nil(t, inputs)
		assert.Nil(t, run)
	})
}

func TestStatusEvaluatorStatusWithLookupInputString(t *testing.T) {
	ctx := context.Background()
	existing := &storage.TaskRunWithID{ID: 9}

	// the first query uses the original inputs, it never finds a run,
	// a later query finds a run only if its digest differs from the first.
	lookupFindsRun := func(found *storage.TaskRunWithID) func(int, string, string, string) (*storage.TaskRunWithID, error) {
		var first string

		return func(n int, _, _, digest string) (*storage.TaskRunWithID, error) {
			if n == 0 {
				first = digest
				return nil, storage.ErrNotExist
			}

			if digest == first {
				return nil, storage.ErrNotExist
			}

			return found, nil
		}
	}

	t.Run("run found via the lookup string returns the lookup inputs", func(t *testing.T) {
		st := &fakeRunLookupStorer{lookup: lookupFindsRun(existing)}

		status, inputs, run, err := newStatusTestEvaluator(st, "lookup").Status(ctx, newStatusTestTask())
		require.NoError(t, err)

		assert.Equal(t, TaskStatusRunExist, status)
		assert.Same(t, existing, run)
		assert.Equal(t, 2, st.calls)

		got := inputStrings(inputs)
		assert.Contains(t, got, "string:lookup")
		assert.NotContains(t, got, "string:orig", "the original input string is replaced by the lookup string")
	})

	t.Run("no run found via the lookup string returns the original inputs", func(t *testing.T) {
		st := &fakeRunLookupStorer{lookup: func(int, string, string, string) (*storage.TaskRunWithID, error) {
			return nil, storage.ErrNotExist
		}}

		status, inputs, run, err := newStatusTestEvaluator(st, "lookup").Status(ctx, newStatusTestTask())
		require.NoError(t, err)

		assert.Equal(t, TaskStatusExecutionPending, status)
		assert.Nil(t, run)
		assert.Equal(t, 2, st.calls, "the lookup string variant is queried too")

		got := inputStrings(inputs)
		assert.Contains(t, got, "string:orig", "a new run must be recorded with the original input string")
		assert.NotContains(t, got, "string:lookup")
	})

	t.Run("an error while querying with the lookup string is returned", func(t *testing.T) {
		boom := errors.New("db down")
		st := &fakeRunLookupStorer{lookup: func(n int, _, _, _ string) (*storage.TaskRunWithID, error) {
			if n == 0 {
				return nil, storage.ErrNotExist
			}
			return nil, boom
		}}

		status, inputs, _, err := newStatusTestEvaluator(st, "lookup").Status(ctx, newStatusTestTask())

		require.ErrorIs(t, err, boom)
		assert.Equal(t, TaskStatusUndefined, status)
		assert.Nil(t, inputs)
	})

	t.Run("the lookup string is not queried if a run exists for the original inputs", func(t *testing.T) {
		st := &fakeRunLookupStorer{lookup: func(int, string, string, string) (*storage.TaskRunWithID, error) {
			return existing, nil
		}}

		status, _, _, err := newStatusTestEvaluator(st, "lookup").Status(ctx, newStatusTestTask())
		require.NoError(t, err)

		assert.Equal(t, TaskStatusRunExist, status)
		assert.Equal(t, 1, st.calls)
	})
}
