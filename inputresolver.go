package baur

import (
	"fmt"
	"path"
	"path/filepath"
	"strings"

	"github.com/simplesurance/baur/v1/cfg"
	"github.com/simplesurance/baur/v1/internal/log"
	"github.com/simplesurance/baur/v1/internal/resolve/gitpath"
	"github.com/simplesurance/baur/v1/internal/resolve/glob"
	"github.com/simplesurance/baur/v1/internal/resolve/gosource"
)

type InputResolver struct {
	gitGlobPathResolver *gitpath.Resolver
	globPathResolver    *glob.Resolver
	goSourceResolver    *gosource.Resolver
}

func NewInputResolver() *InputResolver {
	return &InputResolver{
		gitGlobPathResolver: &gitpath.Resolver{},
		globPathResolver:    &glob.Resolver{},
		goSourceResolver:    gosource.NewResolver(log.Debugf),
	}
}

// Resolves the input definition of the task to concrete Files.
// If an input definition does not resolve to >= paths, an error is returned.
// The resolved Files are deduplicated.
func (i *InputResolver) Resolve(repositoryDir string, task *Task) (*Inputs, error) {
	goSourcePaths, err := i.resolveGoSrcInputs(task.Directory, &task.UnresolvedInputs.GolangSources)
	if err != nil {
		return nil, fmt.Errorf("resolving golang source inputs failed: %w", err)
	}

	gitPaths, err := i.resolveGitGlobPaths(repositoryDir, task.Directory, &task.UnresolvedInputs.GitFiles)
	if err != nil {
		return nil, fmt.Errorf("resolving git-file inputs failed: %w", err)
	}

	globPaths, err := i.resolveGlobPaths(task.Directory, &task.UnresolvedInputs.Files)
	if err != nil {
		return nil, fmt.Errorf("resolving glob file inputs failed: %w", err)
	}

	allInputsPaths := make([]string, 0, len(goSourcePaths)+len(globPaths)+len(gitPaths)+1)
	allInputsPaths = append(allInputsPaths, gitPaths...)
	allInputsPaths = append(allInputsPaths, globPaths...)
	allInputsPaths = append(allInputsPaths, goSourcePaths...)

	// Add the .app.toml file of the app to the inputs
	// TODO: add the files that were included in the .app.toml and it's includes
	allInputsPaths = append(allInputsPaths, path.Join(task.Directory, AppCfgFile))

	uniqFiles, err := i.pathsToUniqFiles(repositoryDir, allInputsPaths)
	if err != nil {
		return nil, err
	}

	return &Inputs{Files: uniqFiles}, nil
}

func (i *InputResolver) resolveGitGlobPaths(repositoryRootDir, appDir string, inputs *cfg.GitFileInputs) ([]string, error) {
	if len(inputs.Paths) == 0 {
		return nil, nil
	}

	gitPaths, err := i.gitGlobPathResolver.Resolve(appDir, inputs.Paths...)
	if err != nil {
		return nil, err
	}

	if len(gitPaths) == 0 {
		return nil, fmt.Errorf("'%s' matched 0 files", strings.Join(inputs.Paths, ", "))
	}

	return gitPaths, err
}

func (i *InputResolver) resolveGlobPaths(appDir string, inputs *cfg.FileInputs) ([]string, error) {
	if len(inputs.Paths) == 0 {
		return nil, nil
	}

	// slice will have at least the same sice then inputs.Path, every globPath must resolve to >1 path
	result := make([]string, 0, len(inputs.Paths))

	for _, path := range inputs.Paths {
		var absGlobPath string

		if filepath.IsAbs(path) {
			absGlobPath = path
		} else {
			absGlobPath = filepath.Join(appDir, path)
		}

		resolvedPaths, err := i.globPathResolver.Resolve(absGlobPath)
		if err != nil {
			return nil, err
		}

		if len(resolvedPaths) == 0 {
			return nil, fmt.Errorf("'%s' matched 0 files", path)
		}

		result = append(result, resolvedPaths...)
	}

	return result, nil
}

func (i *InputResolver) resolveGoSrcInputs(appDir string, inputs *cfg.GolangSources) ([]string, error) {
	if len(inputs.Queries) == 0 && len(inputs.Environment) == 0 {
		return nil, nil
	}

	return i.goSourceResolver.Resolve(appDir, inputs.Environment, inputs.Queries)
}

func (i *InputResolver) pathsToUniqFiles(repositoryRoot string, pathSlice ...[]string) ([]*Inputfile, error) {
	var pathsCount int

	for _, paths := range pathSlice {
		pathsCount += len(paths)
	}

	res := make([]*Inputfile, 0, pathsCount)
	dedupMap := make(map[string]struct{}, pathsCount)

	for _, paths := range pathSlice {
		for _, path := range paths {
			if _, exist := dedupMap[path]; exist {
				log.Debugf("removed duplicate input %q", path)
				continue
			}

			dedupMap[path] = struct{}{}

			relPath, err := filepath.Rel(repositoryRoot, path)
			if err != nil {
				return nil, err
			}

			res = append(res, NewFile(repositoryRoot, relPath))
		}
	}

	return res, nil
}
