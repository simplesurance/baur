package gosource

import (
	"context"
	"errors"
	"fmt"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"golang.org/x/tools/go/packages"

	"github.com/simplesurance/baur/v5/internal/fs"
	"github.com/simplesurance/baur/v5/internal/set"
)

const (
	globQueryPrefix = "fileglob="
	fileQueryPrefix = "file="
)

var defLogFn = func(string, ...any) {}

// Resolver determines all Go Source files that are imported by Go-Files
// in the passed paths
type Resolver struct {
	logFn func(string, ...any)
}

// NewResolver returns a resolver that resolves all go source files in the
// GoDirs and their imports to filepaths.
// env specifies the environment variables to use during resolving.
// If empty or nil the default Go environment is used.
func NewResolver(debugLogFn func(string, ...any)) *Resolver {
	logFn := defLogFn
	if debugLogFn != nil {
		logFn = debugLogFn
	}

	return &Resolver{
		logFn: logFn,
	}
}

func resolveGlobs(workDir string, queries []string) ([]string, error) {
	result := make([]string, 0, len(queries))

	for _, q := range queries {
		if !strings.HasPrefix(q, globQueryPrefix) {
			result = append(result, q)

			continue
		}

		q = strings.TrimPrefix(q, globQueryPrefix)
		q = filepath.Join(workDir, q)

		files, err := fs.FileGlob(q)
		if err != nil {
			return nil, fmt.Errorf("resolving glob %q failed: %w", q, err)
		}

		for _, f := range files {
			result = append(result, fmt.Sprintf("file=%s", f))
		}
	}

	return result, nil
}

func toAbsFilePatternPaths(workDir string, queries []string) {
	for i, q := range queries {
		v, found := strings.CutPrefix(q, fileQueryPrefix)
		if !found || filepath.IsAbs(v) {
			continue
		}

		queries[i] = fileQueryPrefix + filepath.Join(workDir, v)

	}
}

// Resolve resolves queries to Go source file paths and go.mod files
// of the packages and their recursively imported packages.
// Queries must be in go-list query format.
// Testcase files are ignored when false is passed for withTests.
// Files in GOROOT (stdlib packages) and in the GOMODCACHE (non-vendored
// third-party package dependencies) are omitted from the result.
func (r *Resolver) Resolve(ctx context.Context,
	workdir string,
	environment []string,
	buildFlags []string,
	withTests bool,
	queries []string,
) ([]string, error) {
	if len(queries) == 0 {
		return nil, errors.New("queries parameter is empty")
	}

	env := append(whitelistedEnv(), environment...)

	// relative query paths are interpreted as relative to the cwd, not the
	// cfg.WorkDir passed to packages.Resolve
	// (https://github.com/golang/go/issues/65965).
	// As woraround convert them to abs paths manually:
	toAbsFilePatternPaths(workdir, queries)

	queries, err := resolveGlobs(workdir, queries)
	if err != nil {
		return nil, fmt.Errorf("resolving globs in queries failed: %w", err)
	}

	goEnv, err := getGoEnv(workdir, env)
	if err != nil {
		return nil, err
	}

	return r.resolve(ctx, workdir, goEnv, env, buildFlags, withTests, queries)
}

// whitelistedEnvVars returns whitelisted environment variables from the host
// that are set during resolving.
func whitelistedEnv() []string {
	var env []string

	// PATH might be required to locate the "go list" tool on the host
	// system
	if path, exist := os.LookupEnv("PATH"); exist {
		env = append(env, "PATH="+path)
	}

	// The following variables are required for go list to determine the go
	// build cache, see: https://github.com/golang/go/blob/release-branch.go1.11/src/cmd/go/internal/cache/default.go#L112.
	// When those are not set, resolving fails because "go list -compiled" is called internally which requires a gocache dir
	if gocache, exist := os.LookupEnv("GOCACHE"); exist {
		env = append(env, "GOCACHE="+gocache)
	}

	if xdgCacheHome, exist := os.LookupEnv("XDG_CACHE_HOME"); exist {
		env = append(env, "XDG_CACHE_HOME="+xdgCacheHome)
	}

	if home, exist := os.LookupEnv("HOME"); exist {
		env = append(env, "HOME="+home)
	}

	// plan9 home env variable
	if home, exist := os.LookupEnv("home"); exist {
		env = append(env, "home="+home)
	}

	// windows: LocalAppData is used to determine default GOCACHE dir
	if appData, exist := os.LookupEnv("LocalAppData"); exist {
		env = append(env, "LocalAppData="+appData)
	}

	// windows: USERPROFILE is used to determine default GOPATH
	if userprofile, exist := os.LookupEnv("USERPROFILE"); exist {
		env = append(env, "USERPROFILE="+userprofile)
	}

	return env
}

func (r *Resolver) resolve(
	ctx context.Context,
	workdir string,
	goEnv *goEnv,
	env []string,
	buildFlags []string,
	withTests bool,
	queries []string,
) ([]string, error) {
	r.logFn("gosource-resolver: resolving queries: %+v\n"+
		"workdir: %s\n"+
		"env: %+v\n"+
		"goenv: %+v\n"+
		"withTests: %t\n"+
		"buildFlags: %+v\n",
		queries, workdir, env, goEnv, withTests, buildFlags)

	cfg := &packages.Config{
		Context: ctx,
		Mode: packages.NeedName |
			packages.NeedFiles |
			packages.NeedImports |
			packages.NeedEmbedFiles |
			packages.NeedModule,
		Dir:        workdir,
		Env:        env,
		Logf:       r.logFn,
		Tests:      withTests,
		BuildFlags: buildFlags,
	}

	lpkgs, err := packages.Load(cfg, queries...)
	if err != nil {
		return nil, err
	}

	allPkgs := set.Set[*packages.Package]{}
	var visitFn func(*packages.Package)
	visitFn = func(lpkg *packages.Package) {
		if allPkgs.Contains(lpkg) {
			return
		}

		for path := range maps.Keys(lpkg.Imports) {
			visitFn(lpkg.Imports[path])
		}

		allPkgs.Add(lpkg)
	}

	for _, lpkg := range lpkgs {
		visitFn(lpkg)
	}

	var srcFiles []string
	// because multiple packages can be part of the same GoMod,
	// we dedup the go.mod paths with this set:
	gomodFiles := set.Set[string]{}
	for pkg := range allPkgs {
		srcFiles = slices.Concat(srcFiles,
			withoutStdblibAndCacheFiles(goEnv, pkg.GoFiles),
			withoutStdblibAndCacheFiles(goEnv, pkg.OtherFiles),
			withoutStdblibAndCacheFiles(goEnv, pkg.EmbedFiles),
		)

		if len(pkg.Errors) != 0 {
			return nil, fmt.Errorf("resolving source files of package %s failed: %+v", pkg.Name, pkg.Errors)
		}

		if pkg.Module != nil {
			if pkg.Module.Error != nil {
				return nil, fmt.Errorf("loading go module information of package %s failed: %s", pkg.Name, pkg.Module.Error.Err)
			}
			if pkg.Module.GoMod != "" && !isStdLibOrCacheFile(goEnv, pkg.Module.GoMod) {
				gomodFiles.Add(pkg.Module.GoMod)
			}
		}
	}

	return slices.Concat(srcFiles, gomodFiles.Slice()), nil
}

func withoutStdblibAndCacheFiles(env *goEnv, paths []string) []string {
	// likely that more space than needed is allocated, number of allocs
	// is reduced
	result := make([]string, 0, len(paths))
	for _, p := range paths {
		if !isStdLibOrCacheFile(env, p) {
			result = append(result, p)
		}
	}

	return result
}

func isStdLibOrCacheFile(env *goEnv, p string) bool {
	return (len(env.GoCache) > 0 && strings.HasPrefix(p, env.GoCache)) ||
		strings.HasPrefix(p, env.GoRoot) ||
		strings.HasPrefix(p, env.GoModCache)
}
