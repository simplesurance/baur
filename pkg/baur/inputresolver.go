package baur

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/simplesurance/baur/v3/internal/log"
	"github.com/simplesurance/baur/v3/internal/resolve/glob"
	"github.com/simplesurance/baur/v3/internal/resolve/gosource"
	"github.com/simplesurance/baur/v3/internal/vcs"
	"github.com/simplesurance/baur/v3/pkg/cfg"
)

// InputResolver resolves input definitions of a task to concrete files.
type InputResolver struct {
	globPathResolver     *glob.Resolver
	goSourceResolver     *gosource.Resolver
	environmentVariables map[string]string

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

	envVars, err := i.resolveEnvVarInputs(task.UnresolvedInputs.EnvironmentVariables)
	if err != nil {
		return nil, fmt.Errorf("resolving environment variable inputs failed: %w", err)
	}

	stats := i.cache.Statistics()
	log.Debugf("inputresolver: cache statistic: %d entries, %d hits, %d miss, ratio %.2f%%\n",
		stats.Entries, stats.Hits, stats.Miss, stats.HitRatio())

	return append(uniqInputs, envVarMapToInputslice(envVars)...), nil
}

func (i *InputResolver) resolveCacheFileGlob(path string, optional bool) ([]string, error) {
	// resolving files with Optional flag must be handled with care:
	// If optional is true and path does not exist, resolving must not result in an error.
	// If !optional and parts of the path does not exist an error must be returned.
	// We can not use cached results of lookups with optional
	// flag enabled, if an lookup with !optional is requested, it would
	// suppress non-existing path errors.
	// Also only successful !optional must be cached.
	cacheKey := inputResolverFileCacheKey{
		Path:     path,
		Optional: optional,
	}

	if result := i.cache.GetFileInputs(&cacheKey); result != nil {
		return result, nil
	}

	if optional {
		if result := i.cache.GetFileInputs(&inputResolverFileCacheKey{Path: path, Optional: false}); result != nil {
			return result, nil
		}
	}

	result, err := i.globPathResolver.Resolve(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			result = []string{}
			i.cache.AddFileInputs(
				&inputResolverFileCacheKey{Path: path, Optional: true},
				result,
			)

			if optional {
				return result, nil
			}
		}

		return result, err
	}

	i.cache.AddFileInputs(&cacheKey, result)

	return result, err
}

func (i *InputResolver) resolveFileInputs(repositoryDir, appDir string, inputs []cfg.FileInputs) ([]string, error) {
	var result []string

	for _, in := range inputs {
		for _, path := range in.Paths {
			var resolvedPaths []string
			var err error

			if !filepath.IsAbs(path) {
				path = filepath.Join(appDir, path)
			}

			cacheKey := inputResolverFileCacheKey{
				Path:           path,
				GitTrackedOnly: in.GitTrackedOnly,
				Optional:       in.Optional,
			}
			if files := i.cache.GetFileInputs(&cacheKey); files != nil {
				result = append(result, files...)
				continue
			}

			resolvedPaths, err = i.resolveCacheFileGlob(path, in.Optional)
			if err != nil {
				return nil, err
			}

			if len(resolvedPaths) > 0 && in.GitTrackedOnly {
				trackedOnlyPaths, err := i.vcsState.WithoutUntracked(resolvedPaths...)
				if err != nil {
					return nil, fmt.Errorf("removing untracked git files for input %q failed: %w", path, err)
				}
				resolvedPaths = trackedOnlyPaths
			}

			if !in.Optional && len(resolvedPaths) == 0 {
				return nil, fmt.Errorf("'%s' matched 0 files", path)
			}

			i.cache.AddFileInputs(&cacheKey, resolvedPaths)
			result = append(result, resolvedPaths...)
		}
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

func (i *InputResolver) pathsToUniqInputs(repositoryRoot string, paths []string) ([]Input, error) {
	pathsCount := len(paths)

	res := make([]Input, 0, pathsCount)
	dedupMap := make(map[string]struct{}, pathsCount)

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

		res = append(res, i.inputFileSingletonCache.CreateOrGetInputFile(path, relPath))
	}

	return res, nil
}

func (i *InputResolver) setEnvVars() {
	if i.environmentVariables != nil {
		return
	}

	// os.Environ() does not return env variables that are declared but undefined.
	// environment variables that have an empty string assigned are returned.
	environ := os.Environ()
	i.environmentVariables = make(map[string]string, len(environ))

	for _, env := range environ {
		k, v, found := strings.Cut(env, "=")
		if !found {
			// impossible scenario
			panic(fmt.Sprintf("element %q returned by os.Environ() does not contain a '=' character", env))
		}

		i.environmentVariables[k] = v
	}
}

func (i *InputResolver) getEnvVar(namePattern string) (map[string]string, error) {
	const globPatternChars = `*?[]\`

	if !strings.ContainsAny(namePattern, globPatternChars) {
		val, exist := i.environmentVariables[namePattern]
		if !exist {
			return nil, nil
		}

		return map[string]string{namePattern: val}, nil
	}

	res := map[string]string{}
	for k, v := range i.environmentVariables {
		matched, err := path.Match(namePattern, k)
		if err != nil {
			return nil, err
		}
		if matched {
			res[k] = v
		}
	}

	return res, nil
}

func (i *InputResolver) resolveEnvVarInputs(inputs []cfg.EnvVarsInputs) (map[string]string, error) {
	if len(inputs) == 0 {
		return nil, nil
	}

	i.setEnvVars()
	resolvedEnvVars := map[string]string{}

	for _, e := range inputs {
		for _, pattern := range e.Names {
			envVars, err := i.getEnvVar(pattern)
			if err != nil {
				return nil, fmt.Errorf("environment variable name: %q: %w", pattern, err)
			}

			if len(envVars) == 0 && !e.Optional {
				return nil, fmt.Errorf("environment variable %q is undefined", pattern)
			}

			for k, v := range envVars {
				resolvedEnvVars[k] = v
			}
		}
	}

	return resolvedEnvVars, nil
}

func envVarMapToInputslice(envVars map[string]string) []Input {
	res := make([]Input, 0, len(envVars))

	for k, v := range envVars {
		res = append(res, NewInputEnvVar(k, v))
	}

	return res
}
