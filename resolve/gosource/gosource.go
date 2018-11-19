package gosource

import (
	"fmt"
	"go/build"
	"os"
	"path/filepath"
	"strings"

	"github.com/pkg/errors"
	"golang.org/x/tools/go/packages"
)

// Resolver determines all Go Source files that are imported by Go-Files
// in the passed paths
type Resolver struct {
	env    []string
	goDirs []string
}

// NewResolver returns a resolver that resolves all go source files in the
// GoDirs and it's imports to filepaths.
// env specifies the environment variables to use during resolving.
// If empty or nil the default Go environment is used.
func NewResolver(env []string, goDirs ...string) *Resolver {
	return &Resolver{
		env:    env,
		goDirs: goDirs,
	}
}

// Resolve returns the Go source files in the passed directories plus all
// source files of the imported packages.
// Testfiles and stdlib dependencies are ignored.
func (r *Resolver) Resolve() ([]string, error) {
	var allFiles []string
	for _, path := range r.goDirs {
		files, err := r.resolve(path)
		if err != nil {
			return nil, err
		}

		allFiles = append(allFiles, files...)
	}

	return allFiles, nil
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

	return env
}

func (r *Resolver) resolve(path string) ([]string, error) {
	cfg := &packages.Config{
		Mode: packages.LoadImports,
		Dir:  path,
		Env:  append(whitelistedEnv(), r.env...),
	}

	lpkgs, err := packages.Load(cfg, "./...")
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
		err = sourceFiles(&srcFiles, lpkg)
		if err != nil {
			return nil, errors.Wrapf(err, "resolving sourcefiles of package '%s' failed", lpkg.Name)
		}

		if len(lpkg.Errors) != 0 {
			return nil, fmt.Errorf("parsing package %s failed: %+v", lpkg.Name, lpkg.Errors)
		}
	}

	return srcFiles, nil
}

// sourceFiles returns GoFiles and OtherFiles of the package that are not part
// of the stdlib
func sourceFiles(result *[]string, pkg *packages.Package) error {
	err := withoutStdblibPackages(result, pkg.GoFiles)
	if err != nil {
		return err
	}

	err = withoutStdblibPackages(result, pkg.OtherFiles)
	if err != nil {
		return err
	}

	return nil
}

func withoutStdblibPackages(result *[]string, paths []string) error {
	for _, path := range paths {
		abs, err := filepath.Abs(path)
		if err != nil {
			return err
		}

		if strings.HasPrefix(abs, build.Default.GOROOT) {
			continue
		}

		*result = append(*result, abs)
	}

	return nil
}
