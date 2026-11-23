package baur

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/simplesurance/baur/v5/internal/digest"
	"github.com/simplesurance/baur/v5/internal/digest/sha384"
	"github.com/simplesurance/baur/v5/internal/exec"
	"github.com/simplesurance/baur/v5/internal/testutils/gittest"
	"github.com/simplesurance/baur/v5/internal/vcs/git"
	"github.com/simplesurance/baur/v5/pkg/storage"
)

type fakeOutput struct {
	name      string
	typ       OutputType
	size      uint64
	sizeErr   error
	digestErr error
}

func (o *fakeOutput) Name() string               { return o.name }
func (o *fakeOutput) String() string             { return "fake:" + o.name }
func (o *fakeOutput) Exists() (bool, error)      { return true, nil }
func (o *fakeOutput) SizeBytes() (uint64, error) { return o.size, o.sizeErr }
func (o *fakeOutput) Type() OutputType           { return o.typ }
func (o *fakeOutput) Digest() (*digest.Digest, error) {
	if o.digestErr != nil {
		return nil, o.digestErr
	}

	h := sha384.New()
	_ = h.AddBytes([]byte(o.name))

	return h.Digest(), nil
}

type unknownInput struct{}

func (unknownInput) Digest() (*digest.Digest, error) { return sha384.New().Digest(), nil }
func (unknownInput) String() string                  { return "unknown" }

type capturingStorer struct {
	storage.Storer

	saved *storage.TaskRunFull
	id    int
	err   error
}

func (c *capturingStorer) SaveTaskRun(_ context.Context, tr *storage.TaskRunFull) (int, error) {
	c.saved = tr
	return c.id, c.err
}

func TestInputsToStorageInputs(t *testing.T) {
	file := newTestFile(t, "app/f.txt", "hello", false)
	str := NewInputString("hello")
	env := NewInputEnvVar("FOO", "bar")
	task := NewInputTask(&Task{ID: "app.build"}, NewInputs([]Input{str}))

	got, err := inputsToStorageInputs(NewInputs([]Input{file, str, env, task}))
	require.NoError(t, err)

	require.Len(t, got.Files, 1)
	assert.Equal(t, "app/f.txt", got.Files[0].Path)
	assert.Equal(t, goldenFile, got.Files[0].Digest)

	require.Len(t, got.Strings, 1)
	assert.Equal(t, "hello", got.Strings[0].String)
	assert.Equal(t, goldenStringHello, got.Strings[0].Digest)

	require.Len(t, got.EnvironmentVariables, 1)
	assert.Equal(t, "FOO", got.EnvironmentVariables[0].Name)
	assert.Equal(t, goldenEnvVarFooBar, got.EnvironmentVariables[0].Digest)

	require.Len(t, got.TaskInfo, 1)
	assert.Equal(t, "app.build", got.TaskInfo[0].Name)
	assert.NotEmpty(t, got.TaskInfo[0].Digest)
}

func TestInputsToStorageInputsErrors(t *testing.T) {
	t.Run("unsupported input type", func(t *testing.T) {
		_, err := inputsToStorageInputs(NewInputs([]Input{unknownInput{}}))
		require.ErrorContains(t, err, "unsupported input type")
	})

	t.Run("digest error", func(t *testing.T) {
		_, err := inputsToStorageInputs(NewInputs([]Input{NewInputFile("/a/f", "f", false)}))
		require.Error(t, err)
	})

	t.Run("no inputs", func(t *testing.T) {
		got, err := inputsToStorageInputs(NewInputs(nil))
		require.NoError(t, err)
		assert.Empty(t, got.Files)
	})
}

func TestToStorageOutputs(t *testing.T) {
	start := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)
	stop := start.Add(time.Second)

	file := &fakeOutput{name: "app.tar", typ: FileOutput, size: 42}
	img := &fakeOutput{name: "image.id", typ: DockerOutput, size: 1000}

	got, err := toStorageOutputs([]*UploadResult{
		{Output: file, URL: "s3://b/k", Method: UploadMethodS3, Start: start, Stop: stop},
		{Output: img, URL: "reg/img:1", Method: UploadMethodDocker, Start: start, Stop: stop},
		{Output: file, URL: "/mnt/app.tar", Method: UploadMethodFilecopy, Start: start, Stop: stop},
	})
	require.NoError(t, err)
	require.Len(t, got, 2, "uploads of the same output are grouped")

	byName := map[string]*storage.Output{}
	for _, o := range got {
		byName[o.Name] = o
	}

	f := byName["app.tar"]
	require.NotNil(t, f)
	assert.Equal(t, storage.ArtifactTypeFile, f.Type)
	assert.Equal(t, uint64(42), f.SizeBytes)
	assert.NotEmpty(t, f.Digest)
	assert.Equal(t, []*storage.Upload{
		{URI: "s3://b/k", UploadStartTimestamp: start, UploadStopTimestamp: stop, Method: storage.UploadMethodS3},
		{URI: "/mnt/app.tar", UploadStartTimestamp: start, UploadStopTimestamp: stop, Method: storage.UploadMethodFileCopy},
	}, f.Uploads)

	d := byName["image.id"]
	require.NotNil(t, d)
	assert.Equal(t, storage.ArtifactTypeDocker, d.Type)
	assert.Equal(t, uint64(1000), d.SizeBytes)
	require.Len(t, d.Uploads, 1)
	assert.Equal(t, storage.UploadMethodDockerRegistry, d.Uploads[0].Method)
}

func TestToStorageOutputsErrors(t *testing.T) {
	up := func(o Output) []*UploadResult {
		return []*UploadResult{{Output: o, Method: UploadMethodS3}}
	}

	t.Run("no uploads", func(t *testing.T) {
		got, err := toStorageOutputs(nil)
		require.NoError(t, err)
		assert.Empty(t, got)
	})

	t.Run("size error", func(t *testing.T) {
		_, err := toStorageOutputs(up(&fakeOutput{name: "x", sizeErr: errors.New("no size")}))
		require.ErrorContains(t, err, "no size")
	})

	t.Run("digest error", func(t *testing.T) {
		_, err := toStorageOutputs(up(&fakeOutput{name: "x", digestErr: errors.New("no digest")}))
		require.ErrorContains(t, err, "no digest")
	})

	t.Run("unsupported output type", func(t *testing.T) {
		_, err := toStorageOutputs(up(&fakeOutput{name: "x", typ: OutputType(99)}))
		require.ErrorContains(t, err, "unsupported type")
	})
}

func newGitRepoWithFile(t *testing.T) (dir string, repo *git.Repository) {
	t.Helper()

	dir = t.TempDir()
	gittest.CreateRepository(t, dir)
	require.NoError(t, os.WriteFile(filepath.Join(dir, "f.txt"), []byte("1"), 0o600))
	gittest.CommitFilesToGit(t, dir)

	return dir, git.NewRepository(dir)
}

func TestStoreRun(t *testing.T) {
	ctx := context.Background()
	task := &Task{ID: "app.build", AppName: "app", Name: "build"}
	inputs := NewInputs([]Input{NewInputString("hello"), NewInputEnvVar("FOO", "bar")})
	start := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)
	stop := start.Add(2 * time.Second)

	runResult := func(exitCode int) *RunResult {
		return &RunResult{Result: &exec.Result{ExitCode: exitCode}, StartTime: start, StopTime: stop}
	}

	t.Run("stores a successful run", func(t *testing.T) {
		_, repo := newGitRepoWithFile(t)
		st := &capturingStorer{id: 17}

		id, err := StoreRun(ctx, st, repo, task, inputs, runResult(0), []*UploadResult{
			{Output: &fakeOutput{name: "a", typ: FileOutput}, URL: "s3://b/a", Method: UploadMethodS3},
		})
		require.NoError(t, err)
		assert.Equal(t, 17, id)

		require.NotNil(t, st.saved)
		wantCommit, err := repo.CommitID()
		require.NoError(t, err)

		assert.Equal(t, "app", st.saved.ApplicationName)
		assert.Equal(t, "build", st.saved.TaskName)
		assert.Equal(t, wantCommit, st.saved.VCSRevision)
		assert.False(t, st.saved.VCSIsDirty)
		assert.Equal(t, start, st.saved.StartTimestamp)
		assert.Equal(t, stop, st.saved.StopTimestamp)
		assert.Equal(t, storage.ResultSuccess, st.saved.Result)

		wantDigest, err := inputs.Digest()
		require.NoError(t, err)
		assert.Equal(t, wantDigest.String(), st.saved.TotalInputDigest)

		require.Len(t, st.saved.Inputs.Strings, 1)
		require.Len(t, st.saved.Inputs.EnvironmentVariables, 1)
		require.Len(t, st.saved.Outputs, 1)
		assert.Equal(t, "a", st.saved.Outputs[0].Name)
	})

	t.Run("non-zero exit code is stored as failure", func(t *testing.T) {
		_, repo := newGitRepoWithFile(t)
		st := &capturingStorer{}

		_, err := StoreRun(ctx, st, repo, task, inputs, runResult(2), nil)
		require.NoError(t, err)

		assert.Equal(t, storage.ResultFailure, st.saved.Result)
	})

	t.Run("modified tracked file is stored as dirty worktree", func(t *testing.T) {
		dir, repo := newGitRepoWithFile(t)
		require.NoError(t, os.WriteFile(filepath.Join(dir, "f.txt"), []byte("changed"), 0o600))
		st := &capturingStorer{}

		_, err := StoreRun(ctx, st, repo, task, inputs, runResult(0), nil)
		require.NoError(t, err)

		assert.True(t, st.saved.VCSIsDirty)
	})

	t.Run("storage error is returned", func(t *testing.T) {
		_, repo := newGitRepoWithFile(t)
		boom := errors.New("db down")

		_, err := StoreRun(ctx, &capturingStorer{err: boom}, repo, task, inputs, runResult(0), nil)

		require.ErrorIs(t, err, boom)
	})

	t.Run("directory that is not a git repository is an error and nothing is stored", func(t *testing.T) {
		st := &capturingStorer{}

		id, err := StoreRun(ctx, st, git.NewRepository(t.TempDir()), task, inputs, runResult(0), nil)

		require.Error(t, err)
		assert.Equal(t, -1, id)
		assert.Nil(t, st.saved)
	})

	t.Run("input digest error is returned and nothing is stored", func(t *testing.T) {
		_, repo := newGitRepoWithFile(t)
		st := &capturingStorer{}
		bad := NewInputs([]Input{NewInputFile("/a/f", "f", false)})

		id, err := StoreRun(ctx, st, repo, task, bad, runResult(0), nil)

		require.Error(t, err)
		assert.Equal(t, -1, id)
		assert.Nil(t, st.saved)
	})
}

// The order of the stored outputs decides the IDs the database assigns to
// them, it must not change between runs of the same task.
func TestToStorageOutputsKeepsTheOrderOfTheFirstUpload(t *testing.T) {
	names := []string{"o0", "o1", "o2", "o3", "o4", "o5", "o6", "o7", "o8", "o9"}

	uploads := make([]*UploadResult, 0, len(names))
	for _, n := range names {
		uploads = append(uploads, &UploadResult{Output: &fakeOutput{name: n, typ: FileOutput}, Method: UploadMethodS3})
	}

	for range 20 {
		got, err := toStorageOutputs(uploads)
		require.NoError(t, err)

		gotNames := make([]string, 0, len(got))
		for _, o := range got {
			gotNames = append(gotNames, o.Name)
		}

		require.Equal(t, names, gotNames)
	}
}
