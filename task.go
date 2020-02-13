package baur

import (
	"fmt"
	"path"
	"path/filepath"
	"sort"
	"strings"

	"github.com/simplesurance/baur/cfg"
	"github.com/simplesurance/baur/digest"
	"github.com/simplesurance/baur/digest/sha384"
	"github.com/simplesurance/baur/fs"
	"github.com/simplesurance/baur/log"
	"github.com/simplesurance/baur/resolve/gitpath"
	"github.com/simplesurance/baur/resolve/glob"
	"github.com/simplesurance/baur/resolve/gosource"
	"github.com/simplesurance/baur/upload/scheduler"
)

// Task is a an execution step belonging to an app.
// A task has a set of Inputs that produce a set of outputs by executing it's
// Command.
type Task struct {
	RepositoryRoot string
	Directory      string

	AppName string

	Name             string
	Command          string
	UnresolvedInputs *cfg.Input
	resolvedInputs   []*File
	totalInputDigest *digest.Digest
	Outputs          *cfg.Output
}

// NewTask returns a new Task.
func NewTask(cfg *cfg.Task, appName, repositoryRootdir, workingDir string) *Task {
	return &Task{
		RepositoryRoot:   repositoryRootdir,
		Directory:        workingDir,
		Outputs:          &cfg.Output,
		Command:          cfg.Command,
		Name:             cfg.Name,
		AppName:          appName,
		UnresolvedInputs: &cfg.Input,
	}
}

// ID returns <APP-NAME>.<TASK-NAME>
func (t *Task) ID() string {
	return fmt.Sprintf("%s.%s", t.AppName, t.Name)
}

// String returns ID()
func (t *Task) String() string {
	return t.ID()
}

// HasInputs returns true if Inputs are defined for the app
func (t *Task) HasInputs() bool {
	return !cfg.InputsAreEmpty(t.UnresolvedInputs)
}

// TotalInputDigest returns the total input digest that is calculated over all
// input sources. The calculation is only done on the 1. call on following calls
// the stored digest is returned.
func (t *Task) TotalInputDigest() (digest.Digest, error) {
	if t.totalInputDigest != nil {
		return *t.totalInputDigest, nil
	}

	buildInputs, err := t.Inputs()
	if err != nil {
		return digest.Digest{}, err
	}

	digests := make([]*digest.Digest, 0, len(buildInputs))
	for _, bi := range buildInputs {
		d, err := bi.Digest()
		if err != nil {
			return digest.Digest{}, fmt.Errorf("calculating input digest of %q failed: %w", bi, err)
		}

		digests = append(digests, &d)
	}

	totalDigest, err := sha384.Sum(digests)
	if err != nil {
		return digest.Digest{}, fmt.Errorf("calculating total input digest: %w", err)
	}

	t.totalInputDigest = totalDigest

	return *t.totalInputDigest, nil
}

// Inputs resolves and returns all inputs of the task.
// The Inputs are deduplicated before they are returned.
// If one more resolved path does not match a file an error is generated.
// If not build inputs are defined, an empty slice and no error is returned.
// If the function is called the first time, the BuildInputPaths are resolved
// and stored. On following calls the stored BuildInputs are returned.
func (t *Task) Inputs() ([]*File, error) {
	if t.resolvedInputs != nil {
		return t.resolvedInputs, nil
	}

	paths, err := t.resolveBuildInputPaths()
	if err != nil {
		return nil, err
	}

	t.resolvedInputs, err = t.pathsToUniqFiles(t.RepositoryRoot, paths)
	if err != nil {
		return nil, err
	}

	return t.resolvedInputs, nil
}

func (t *Task) resolveBuildInputPaths() ([]string, error) {
	globPaths, err := t.resolveGlobFileInputs()
	if err != nil {
		return nil, fmt.Errorf("resolving File BuildInputs failed: %w", err)
	}

	gitPaths, err := t.resolveGitFileInputs()
	if err != nil {
		return nil, fmt.Errorf("resolving GitFile BuildInputs failed: %w", err)
	}

	goSrcPaths, err := t.resolveGoSrcInputs()
	if err != nil {
		return nil, fmt.Errorf("resolving GoLangSources BuildInputs failed: %w", err)
	}

	paths := make([]string, 0, len(globPaths)+len(gitPaths)+len(goSrcPaths)+1)
	paths = append(paths, globPaths...)
	paths = append(paths, gitPaths...)
	paths = append(paths, goSrcPaths...)

	// Add the .app.toml file of the app to the inputs
	paths = append(paths, path.Join(t.Directory, AppCfgFile))

	return paths, nil
}

func (t *Task) resolveGlobFileInputs() ([]string, error) {
	var res []string

	for _, globPath := range t.UnresolvedInputs.Files.Paths {
		if !filepath.IsAbs(globPath) {
			globPath = filepath.Join(t.Directory, globPath)
		}

		resolver := glob.NewResolver(globPath)
		paths, err := resolver.Resolve()
		if err != nil {
			return nil, fmt.Errorf("%s: %w", globPath, err)
		}

		if len(paths) == 0 {
			return nil, fmt.Errorf("'%s' matched 0 files", globPath)
		}

		res = append(res, paths...)
	}

	return res, nil
}

func (t *Task) resolveGitFileInputs() ([]string, error) {
	var res []string

	if len(t.UnresolvedInputs.GitFiles.Paths) == 0 {
		return res, nil
	}

	resolver := gitpath.NewResolver(t.Directory, t.UnresolvedInputs.GitFiles.Paths...)
	paths, err := resolver.Resolve()
	if err != nil {
		return nil, err
	}

	if len(paths) == 0 {
		return nil, fmt.Errorf("'%s' matched 0 files", strings.Join(paths, ", "))
	}

	res = append(res, paths...)

	return res, nil
}

func (t *Task) resolveGoSrcInputs() ([]string, error) {
	var res []string

	if len(t.UnresolvedInputs.GolangSources.Paths) == 0 {
		return res, nil
	}

	absGoSourcePaths := fs.AbsPaths(t.Directory, t.UnresolvedInputs.GolangSources.Paths)

	// TODO use a logger instance
	resolver := gosource.NewResolver(log.Debugf, t.UnresolvedInputs.GolangSources.Environment, absGoSourcePaths...)
	paths, err := resolver.Resolve()
	if err != nil {
		return nil, err
	}

	if len(paths) == 0 {
		return nil, fmt.Errorf("'%s' matched 0 files", strings.Join(paths, ", "))
	}

	res = append(res, paths...)

	return res, nil
}

func (t *Task) pathsToUniqFiles(workingDir string, paths []string) ([]*File, error) {
	dedupMap := make(map[string]struct{}, len(paths))
	res := make([]*File, 0, len(paths))

	for _, path := range paths {
		if _, exist := dedupMap[path]; exist {
			// TODO use a logger instance
			log.Debugf("%s: removed duplicate Build Input '%s'", t.ID(), path)
			continue
		}
		dedupMap[path] = struct{}{}

		relPath, err := filepath.Rel(workingDir, path)
		if err != nil {
			return nil, err
		}

		// TODO: should resolving the relative path be done in
		// Newfile() instead?
		res = append(res, NewFile(workingDir, relPath))
	}

	return res, nil
}

// TODO: rename this function when the db and commands support multiple tasks.
// BuildOutputs returns a list of outputs that the task.Command is expected to produce.
func (t *Task) BuildOutputs() ([]BuildOutput, error) {
	result := make([]BuildOutput, 0, len(t.Outputs.DockerImage)+len(t.Outputs.File))

	taskRelDir, err := filepath.Rel(t.Directory, t.RepositoryRoot)
	if err != nil {
		return nil, err
	}

	for _, di := range t.Outputs.DockerImage {
		result = append(result, &DockerArtifact{
			ImageIDFile: path.Join(t.Directory, di.IDFile),
			Tag:         di.RegistryUpload.Tag,
			Repository:  di.RegistryUpload.Repository,
			Registry:    di.RegistryUpload.Registry,
		})
	}

	for _, f := range t.Outputs.File {
		filePath := f.Path
		absFilePath := path.Join(t.Directory, filePath)

		if !f.S3Upload.IsEmpty() {
			url := "s3://" + f.S3Upload.Bucket + "/" + f.S3Upload.DestFile

			result = append(result, &FileArtifact{
				RelPath:   path.Join(taskRelDir, filePath),
				Path:      absFilePath,
				DestFile:  f.S3Upload.DestFile,
				UploadURL: url,
				uploadJob: &scheduler.S3Job{
					DestURL:  url,
					FilePath: absFilePath,
				},
			})
		}

		if !f.FileCopy.IsEmpty() {
			result = append(result, &FileArtifact{
				RelPath:   path.Join(taskRelDir, filePath),
				Path:      absFilePath,
				DestFile:  f.FileCopy.Path,
				UploadURL: f.FileCopy.Path,
				uploadJob: &scheduler.FileCopyJob{
					Src: absFilePath,
					Dst: f.FileCopy.Path,
				},
			})
		}
	}

	return result, nil
}

func SortTasksByID(tasks []*Task) {
	sort.Slice(tasks, func(i int, j int) bool {
		return tasks[i].ID() < tasks[i].ID()
	})
}
