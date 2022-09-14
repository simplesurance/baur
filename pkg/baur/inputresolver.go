package baur

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/simplesurance/baur/v2/internal/log"
	"github.com/simplesurance/baur/v2/internal/resolve/glob"
	"github.com/simplesurance/baur/v2/internal/resolve/gosource"
	"github.com/simplesurance/baur/v2/internal/vcs"
	"github.com/simplesurance/baur/v2/pkg/cfg"
)

// InputResolver resolves input definitions of a task to concrete files.
type InputResolver struct {
	globPathResolver *glob.Resolver
	goSourceResolver *gosource.Resolver

	vcsState                vcs.StateFetcher
	cache                   *inputResolverCache
	inputFileSingletonCache *InputFileSingletonCache
}

// NewInputResolver returns an InputResolver that caches resolver
// results.
func NewInputResolver(vcsState vcs.StateFetcher) *InputResolver {
	return &InputResolver{
		globPathResolver:        &glob.Resolver{},
		goSourceResolver:        gosource.NewResolver(log.Debugf),
		vcsState:                vcsState,
		cache:                   newInputResolverCache(),
		inputFileSingletonCache: NewInputFileSingletonCache(),
	}
}

// Resolve resolves the input definition of the task to concrete Files.
// If an input definition does not resolve to >=1 paths, an error is returned.
// The resolved Files are deduplicated.
func (i *InputResolver) Resolve(ctx context.Context, repositoryDir string, task *Task) ([]Input, error) {
	goSourcePaths, err := i.resolveGoSrcInputs(ctx, task.Directory, task.UnresolvedInputs.GolangSources)
	if err != nil {
		return nil, fmt.Errorf("resolving golang source inputs failed: %w", err)
	}

	globPaths, err := i.resolveFileInputs(repositoryDir, task.Directory, task.UnresolvedInputs.Files)
	if err != nil {
		return nil, fmt.Errorf("resolving file inputs failed: %w", err)
	}

	allInputsPaths := make([]string, 0, len(goSourcePaths)+len(globPaths)+len(task.CfgFilepaths))
	allInputsPaths = append(allInputsPaths, globPaths...)
	allInputsPaths = append(allInputsPaths, goSourcePaths...)

	allInputsPaths = append(allInputsPaths, task.CfgFilepaths...)

	uniqInputs, err := i.pathsToUniqInputs(repositoryDir, allInputsPaths)
	if err != nil {
		return nil, err
	}

	stats := i.cache.Statistics()
	log.Debugf("inputresolver: cache statistic: %d entries, %d hits, %d miss, ratio %.2f%%\n",
		stats.Entries, stats.Hits, stats.Miss, stats.HitRatio())

	return uniqInputs, nil
}

func (i *InputResolver) resolveFileInputs(repositoryDir, appDir string, inputs []cfg.FileInputs) ([]string, error) {
	var result []string

	for _, in := range inputs {
		if files := i.cache.GetFileInputs(appDir, &in); files != nil {
			result = append(result, files...)
			continue
		}

		var files []string
		for _, path := range in.Paths {
			var resolvedPaths []string
			var err error

			if !filepath.IsAbs(path) {
				path = filepath.Join(appDir, path)
			}

			resolvedPaths, err = i.globPathResolver.Resolve(path)
			if err != nil {
				if in.Optional && errors.Is(err, os.ErrNotExist) {
					continue
				}

				return nil, err
			}

			if in.GitTrackedOnly {
				trackedOnlyPaths, err := i.vcsState.WithoutUntracked(resolvedPaths...)
				if err != nil {
					return nil, fmt.Errorf("removing untracked git files for input %q failed: %w", path, err)
				}
				resolvedPaths = trackedOnlyPaths
			}

			if !in.Optional && len(resolvedPaths) == 0 {
				return nil, fmt.Errorf("'%s' matched 0 files", path)
			}

			files = append(files, resolvedPaths...)
		}

		i.cache.AddFileInputs(appDir, &in, files)
		result = append(result, files...)
	}

	return result, nil
}

func (i *InputResolver) resolveGoSrcInputs(ctx context.Context, appDir string, inputs []cfg.GolangSources) ([]string, error) {
	var result []string

	for _, gs := range inputs {
		if files := i.cache.GetGolangSources(appDir, &gs); files != nil {
			result = append(result, files...)
			continue
		}

		files, err := i.goSourceResolver.Resolve(ctx, appDir, gs.Environment, gs.BuildFlags, gs.Tests, gs.Queries)
		if err != nil {
			return nil, err
		}

		i.cache.AddGolangSources(appDir, &gs, files)
		result = append(result, files...)
	}

	return result, nil
}

func (i *InputResolver) pathsToUniqInputs(repositoryRoot string, pathSlice ...[]string) ([]Input, error) {
	var pathsCount int

	for _, paths := range pathSlice {
		pathsCount += len(paths)
	}

	res := make([]Input, 0, pathsCount)
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

			res = append(res, i.inputFileSingletonCache.CreateOrGetInputFile(repositoryRoot, relPath))
		}
	}

	return res, nil
}
