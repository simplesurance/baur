package gosource

import (
	"fmt"
	"go/build"
	"os"
	"path/filepath"
	"strings"

	"github.com/pkg/errors"
	"golang.org/x/tools/go/packages"

	"github.com/simplesurance/baur/exec"
	"github.com/simplesurance/baur/fs"
)

var defLogFn = func(string, ...interface{}) {}

// Resolver determines all Go Source files that are imported by Go-Files
// in the passed paths
type Resolver struct {
	logFn func(string, ...interface{})
}

// NewResolver returns a resolver that resolves all go source files in the
// GoDirs and it's imports to filepaths.
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

// GOROOT runs "go env GOROOT" to determine the GOROOT and returns it.
func GOROOT() (string, error) {
	res, err := exec.Command("go", "env", "GOROOT").ExpectSuccess().Run()
	if err != nil {
		return "", err
	}

	goroot := strings.TrimSpace(res.StrOutput())
	if goroot == "" {
		return "", fmt.Errorf("%s did not print anything", res.Command)
	}

	return goroot, nil
}

// getEnvValue iterates in reverse order through env and returns the value of
// the first found environment variable with the given key.
// If no environment variable with the key is found, an empty string is returned.
func getEnvValue(env []string, key string) string {
	for i := len(env) - 1; i >= 0; i-- {
		idx := strings.Index(env[i], key+"=")
		if idx != -1 {
			return env[i][idx:]
		}
	}

	return ""
}

// Resolve returns the Go source files in the passed directories plus all
// source files of the imported packages.
// Testfiles and stdlib dependencies are ignored.
func (r *Resolver) Resolve(environment []string, directories ...string) ([]string, error) {
	var allFiles []string
	var err error

	if len(directories) == 0 {
		return []string{}, nil
	}

	env := append(whitelistedEnv(), environment...)
	goroot := getEnvValue(env, "GOROOT")

	if goroot == "" {
		goroot, err = GOROOT()
		if err != nil {
			return nil, err
		}

		env = append(env, "GOROOT="+goroot)
	}

	if err := fs.DirsExist(goroot); err != nil {
		return nil, fmt.Errorf(
			"GOROOT directory '%s' does not exist, ensure the GOROOT env variable is set correctly or 'go env root' returns the right path",
			build.Default.GOROOT,
		)
	}

	r.logFn("gosource-resolver: environment: '%s'\n", env)

	for _, path := range directories {
		files, err := r.resolve(path, goroot, env)
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

func (r *Resolver) resolve(path, goroot string, env []string) ([]string, error) {
	cfg := &packages.Config{
		Mode: packages.LoadImports,
		Dir:  path,
		Env:  env,
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
		err = sourceFiles(&srcFiles, goroot, lpkg)
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
func sourceFiles(result *[]string, goroot string, pkg *packages.Package) error {
	err := withoutStdblibPackages(result, goroot, pkg.GoFiles)
	if err != nil {
		return err
	}

	err = withoutStdblibPackages(result, goroot, pkg.OtherFiles)
	if err != nil {
		return err
	}

	return nil
}

func withoutStdblibPackages(result *[]string, goroot string, paths []string) error {
	for _, path := range paths {
		abs, err := filepath.Abs(path)
		if err != nil {
			return err
		}

		if strings.HasPrefix(abs, goroot) {
			continue
		}

		*result = append(*result, abs)
	}

	return nil
}
