package baur

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/simplesurance/baur/v3/internal/digest"
	"github.com/simplesurance/baur/v3/internal/digest/gitobjectid"
	"github.com/simplesurance/baur/v3/internal/digest/sha384"
	"github.com/simplesurance/baur/v3/internal/fs"
	"github.com/simplesurance/baur/v3/internal/log"
	"github.com/simplesurance/baur/v3/internal/resolve/glob"
	"github.com/simplesurance/baur/v3/internal/resolve/gosource"
	"github.com/simplesurance/baur/v3/internal/vcs"
	"github.com/simplesurance/baur/v3/internal/vcs/git"
	"github.com/simplesurance/baur/v3/pkg/cfg"
)

// goSourceResolver returns a list source files required to compile a golang
// binary.
type goSourceResolver interface {
	Resolve(
		ctx context.Context,
		workdir string,
		environment []string,
		buildFlags []string,
		withTests bool,
		queries []string,
	) ([]string, error)
}

// InputResolver resolves input definitions of a task to concrete files.
type InputResolver struct {
	repoDir                 string
	globPathResolver        *glob.Resolver
	goSourceResolver        goSourceResolver
	environmentVariables    map[string]string
	vcsState                vcs.StateFetcher
	inputFileSingletonCache *InputFileSingletonCache
	cache                   *inputResolverCache
	gitTrackedDb            *git.TrackedObjects
	fileHashfn              FileHashFn
}

// NewInputResolver returns an InputResolver that caches resolver
// results.
func NewInputResolver(vcsState vcs.StateFetcher, repoDir string, hashGitUntrackedFiles bool) *InputResolver {
	result := InputResolver{
		repoDir:                 repoDir,
		globPathResolver:        &glob.Resolver{},
		goSourceResolver:        gosource.NewResolver(log.Debugf),
		vcsState:                vcsState,
		cache:                   newInputResolverCache(),
		inputFileSingletonCache: NewInputFileSingletonCache(),
	}

	if _, gitUnavail := vcsState.(*vcs.NoVCsState); gitUnavail {
		result.fileHashfn = sha384.File
		log.Debugf("inputresolver: using sha384 file hasher\n")
		return &result
	}

	result.gitTrackedDb = git.NewTrackedObjects(repoDir, log.Debugf)
	g := gitobjectid.New(repoDir, log.Debugf)
	log.Debugf("inputresolver: using gitobject file hasher\n")

	if hashGitUntrackedFiles {
		result.fileHashfn = g.File
		return &result
	}

	return &result
}

// Resolve resolves the input definition of the task to concrete Files.
// If an input definition does not resolve to >=1 paths, an error is returned.
// The resolved Files are deduplicated.
func (i *InputResolver) Resolve(ctx context.Context, task *Task) ([]Input, error) {
	goSourcePaths, err := i.resolveGoSrcInputs(ctx, task.Directory, task.UnresolvedInputs.GolangSources)
	if err != nil {
		return nil, fmt.Errorf("resolving golang source inputs failed: %w", err)
	}

	globPaths, err := i.resolveFileInputs(task.Directory, task.UnresolvedInputs.Files)
	if err != nil {
		return nil, fmt.Errorf("resolving file inputs failed: %w", err)
	}

	allInputsPaths := make([]string, 0, len(goSourcePaths)+len(globPaths)+len(task.CfgFilepaths))
	allInputsPaths = append(allInputsPaths, globPaths...)
	allInputsPaths = append(allInputsPaths, goSourcePaths...)
	allInputsPaths = append(allInputsPaths, task.CfgFilepaths...)

	uniqInputs, err := i.pathsToUniqInputs(allInputsPaths, fs.AbsPaths(task.Directory, task.UnresolvedInputs.ExcludedFiles.Paths))
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

func (i *InputResolver) resolveFileInputs(appDir string, inputs []cfg.FileInputs) ([]string, error) {
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

func (i *InputResolver) pathsToUniqInputs(paths, excludePatterns []string) ([]Input, error) {
	pathsCount := len(paths)

	res := make([]Input, 0, pathsCount)
	dedupMap := make(map[string]struct{}, pathsCount)

	for _, path := range paths {
		if _, exist := dedupMap[path]; exist {
			log.Debugf("removed duplicate input %q", path)
			continue
		}

		dedupMap[path] = struct{}{}

		excluded, excludePattern, err := i.globPathResolver.Matches(path, excludePatterns)
		if err != nil {
			return nil, fmt.Errorf("ExcludedFiles: %w", err)
		}

		if excluded {
			log.Debugf("removed input %q, matches exclude pattern %q", path, excludePattern)
			continue
		}

		relPath, err := filepath.Rel(i.repoDir, path)
		if err != nil {
			return nil, err
		}

		f, err := i.createOrGetCachedInputFile(context.TODO(), path, relPath)
		if err != nil {
			return nil, err
		}

		res = append(res, f)
	}

	return res, nil
}

func (i *InputResolver) createOrGetCachedInputFile(ctx context.Context, absPath, relPath string) (*InputFile, error) {
	if f, exists := i.inputFileSingletonCache.Get(absPath); exists {
		return f, nil
	}

	if i.gitTrackedDb == nil {
		f, err := i.newInputFile(absPath, relPath)
		if err != nil {
			return nil, err
		}
		return i.inputFileSingletonCache.Add(f), nil
	}

	obj, err := i.gitTrackedDb.Get(ctx, absPath)
	if err == nil {
		f, err := i.newInputFileWithTrackedOjb(ctx, absPath, relPath, obj)
		if err != nil {
			return nil, err
		}
		return i.inputFileSingletonCache.Add(f), nil
	}

	if errors.Is(err, git.ErrObjectNotFound) && i.fileHashfn != nil {
		f, err := i.newInputFile(absPath, relPath)
		if err != nil {
			return nil, err
		}
		return i.inputFileSingletonCache.Add(f), nil
	}

	return nil, err
}

func (i *InputResolver) newInputFile(absPath, relPath string) (*InputFile, error) {
	lfi, err := os.Lstat(absPath)
	if err != nil {
		return nil, err
	}

	if lfi.Mode()&os.ModeSymlink == os.ModeSymlink {
		relTargetPath, err := fs.RealPathRel(i.repoDir, absPath)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", relPath, err)
		}

		executable, err := fs.FileHasOwnerExecPerm(absPath)
		// FileHasOwnerExecPerm is only implemented on Unix, on other
		// platforms it returns ErrUnsupported.
		if err != nil && err != errors.ErrUnsupported { //nolint: errorlint // errors.Is() not needed here and more expensive
			return nil, fmt.Errorf("%s: determining if owner has exec permissions failed %w", absPath, err)
		}

		return NewInputFile(absPath, relPath,
			executable,
			WithHashFn(i.fileHashfn),
			WithSymlinkTargetPath(relTargetPath),
		), nil
	}

	return NewInputFile(absPath, relPath, fs.OwnerHasExecPerm(lfi.Mode()), WithHashFn(i.fileHashfn)), nil
}

func (i *InputResolver) newInputFileWithTrackedOjb(ctx context.Context, absPath, relPath string, obj *git.TrackedObject) (*InputFile, error) {
	if obj.Mode&git.ObjectTypeSymlink == git.ObjectTypeFile {
		return NewInputFile(absPath, relPath,
			fs.OwnerHasExecPerm(os.FileMode(obj.Mode)),
			WithHashFn(i.fileHashfn),
			WithContentDigest(&digest.Digest{Sum: []byte(obj.ObjectID), Algorithm: digest.GitObjectID}),
		), nil
	}

	if obj.Mode&git.ObjectTypeSymlink == git.ObjectTypeSymlink {
		targetPath, err := fs.RealPath(absPath)
		if err != nil {
			return nil, err
		}

		relTargetPath, err := filepath.Rel(i.repoDir, targetPath)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", absPath, err)
		}

		targetObj, err := i.gitTrackedDb.Get(ctx, targetPath)
		if err != nil {
			if errors.Is(err, git.ErrObjectNotFound) && i.fileHashfn != nil {
				return i.newInputFile(absPath, relPath)
			}
			return nil, err
		}

		return NewInputFile(absPath, relPath,
			fs.OwnerHasExecPerm(os.FileMode(targetObj.Mode)),
			WithHashFn(i.fileHashfn),
			WithSymlinkTargetPath(relTargetPath),
			WithContentDigest(&digest.Digest{Sum: []byte(targetObj.ObjectID), Algorithm: digest.GitObjectID}),
		), nil
	}

	return nil, fmt.Errorf("%s: got unsupport git.TrackedObject mode: %o", relPath, obj.Mode)
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
