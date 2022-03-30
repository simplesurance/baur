package gosource

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/tools/go/packages"

	"github.com/simplesurance/baur/v2/internal/fs"
)

const globQueryPrefix = "fileglob="

var defLogFn = func(string, ...interface{}) {}

// Resolver determines all Go Source files that are imported by Go-Files
// in the passed paths
type Resolver struct {
	logFn func(string, ...interface{})
}

// NewResolver returns a resolver that resolves all go source files in the
// GoDirs and their imports to filepaths.
// env specifies the environment variables to use during resolving.
// If empty or nil the default Go environment is used.
func NewResolver(debugLogFn func(string, ...interface{})) *Resolver {
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
			f, err := filepath.Rel(workDir, f)
			if err != nil {
				return nil, fmt.Errorf("resolving glob %q failed: %w", q, err)
			}

			result = append(result, fmt.Sprintf("file=%s", f))
		}
	}

	return result, nil
}

// Resolve returns the Go source files in the passed directories plus all
// source files of the imported packages.
// Testfiles and stdlib dependencies are ignored.
func (r *Resolver) Resolve(
	ctx context.Context,
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

	queries, err := resolveGlobs(workdir, queries)
	if err != nil {
		return nil, fmt.Errorf("resolving globs in queries failed: %w", err)
	}

	goEnv, err := getGoEnv(env)
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
		Context:    ctx,
		Mode:       packages.NeedName | packages.NeedFiles | packages.NeedImports | packages.NeedEmbedFiles,
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

	// We can't use packages.All because
	// we need an ordered traversal.
	var all []*packages.Package // postorder
	seen := make(map[*packages.Package]bool)
	var visit func(*packages.Package)
	visit = func(lpkg *packages.Package) {
		if !seen[lpkg] {
			seen[lpkg] = true

			// visit imports
			var importPaths []string
			for path := range lpkg.Imports {
				importPaths = append(importPaths, path)
			}
			for _, path := range importPaths {
				visit(lpkg.Imports[path])
			}

			all = append(all, lpkg)
		}
	}
	for _, lpkg := range lpkgs {
		visit(lpkg)
	}
	lpkgs = all

	var srcFiles []string
	for _, lpkg := range lpkgs {
		err = sourceFiles(&srcFiles, goEnv, lpkg)
		if err != nil {
			return nil, fmt.Errorf("resolving sourcefiles of package '%s' failed: %w", lpkg.Name, err)
		}

		if len(lpkg.Errors) != 0 {
			return nil, fmt.Errorf("parsing package %s failed: %+v", lpkg.Name, lpkg.Errors)
		}
	}

	return srcFiles, nil
}

// sourceFiles returns GoFiles and OtherFiles of the package that are not part
// of the stdlib
func sourceFiles(result *[]string, env *goEnv, pkg *packages.Package) error {
	err := withoutStdblibAndCacheFiles(result, env, pkg.GoFiles)
	if err != nil {
		return err
	}

	err = withoutStdblibAndCacheFiles(result, env, pkg.OtherFiles)
	if err != nil {
		return err
	}

	err = withoutStdblibAndCacheFiles(result, env, pkg.EmbedFiles)
	if err != nil {
		return err
	}

	return nil
}

func withoutStdblibAndCacheFiles(result *[]string, env *goEnv, paths []string) error {
	for _, path := range paths {
		abs, err := filepath.Abs(path)
		if err != nil {
			return err
		}

		if len(env.GoCache) > 0 && strings.HasPrefix(abs, env.GoCache) {
			continue
		}

		if strings.HasPrefix(abs, env.GoRoot) {
			continue
		}

		// use HasPrefix() + Replace() to ensure we only replace the
		// path if is the prefix
		if strings.HasPrefix(abs, env.GoModCache) {
			abs = strings.Replace(abs, env.GoModCache, "$GOMODCACHE", 1)
		}

		*result = append(*result, abs)
	}

	return nil
}
