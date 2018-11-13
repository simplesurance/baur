package gosource

import (
	"fmt"
	"go/build"
	"os"
	"strings"

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
		env:    append(os.Environ(), env...),
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

func (r *Resolver) resolve(path string) ([]string, error) {
	cfg := &packages.Config{
		Mode: packages.LoadImports,
		Dir:  path,
		Env:  r.env,
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
		srcFiles = append(srcFiles, sourceFiles(lpkg)...)

		if len(lpkg.Errors) != 0 {
			return nil, fmt.Errorf("parsing package %s failed: %+v", lpkg.Name, lpkg.Errors)
		}
	}

	return srcFiles, nil
}

// sourceFiles returns GoFiles and OtherFiles of the package that are not part
// of the stdlib
func sourceFiles(pkg *packages.Package) []string {
	paths := make([]string, 0, len(pkg.GoFiles)+len(pkg.OtherFiles))

	for _, path := range pkg.GoFiles {
		if isStdLib(path) {
			continue
		}

		paths = append(paths, path)
	}

	for _, path := range pkg.OtherFiles {
		if isStdLib(path) {
			continue
		}

		paths = append(paths, path)
	}

	return paths
}

func isStdLib(path string) bool {
	return strings.HasPrefix(path, build.Default.GOROOT)
}
