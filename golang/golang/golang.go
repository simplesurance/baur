// Package golang determines all Go Source files that are imported by a Go-Files
// in a directory.
// Most of the code is based on a slightly modified version of https://github.com/rogpeppe/showdeps
package golang

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/kisielk/gotool"
	"github.com/pkg/errors"
	"github.com/rogpeppe/godeps/build"
)

// SrcFiles returns the Go source files in the passed directories plus all
// source files of the imported packages.
// Testfiles and stdlib dependencies are ignored.
func SrcFiles(gopath string, dirs ...string) ([]string, error) {
	var allFiles []string
	ctx := build.Default

	if len(gopath) > 0 {
		ctx.GOPATH = gopath
	}

	for _, d := range dirs {
		files, err := resolve(ctx, d)
		if err != nil {
			return nil, err
		}

		allFiles = append(allFiles, files...)
	}

	return allFiles, nil
}

func resolve(ctx build.Context, path string) ([]string, error) {
	recur := true
	pkgs := []string{"./..."}

	if err := os.Chdir(path); err != nil {
		return nil, errors.Wrapf(err, "changing cwd to %q failed", path)
	}

	pkgs = gotool.ImportPaths(pkgs)

	rootPkgs := make(map[string]bool)
	for _, pkg := range pkgs {
		p, err := ctx.Import(pkg, path, build.FindOnly)
		if err != nil {
			return nil, errors.Wrapf(err, "cannot find %q", pkg)
		}

		rootPkgs[p.ImportPath] = true
	}

	allPkgs := make(map[string][]string)
	for pkg := range rootPkgs {
		if err := findImports(ctx, pkg, path, recur, allPkgs, rootPkgs); err != nil {
			return nil, errors.Wrapf(err, "cannot find imports from %q", pkg)
		}
	}

	files := make([]string, 0, len(allPkgs))
	for pkgName := range allPkgs {
		pkg, err := ctx.Import(pkgName, path, 0)
		if err != nil {
			return nil, errors.Wrapf(err, "determining imports from %q failed", pkg)
		}

		gofiles := absFilePaths(pkg, pkg.GoFiles)
		cgofiles := absFilePaths(pkg, pkg.CgoFiles)

		files = append(files, gofiles...)
		files = append(files, cgofiles...)
	}

	return files, nil
}

func absFilePaths(pkg *build.Package, fs []string) []string {
	res := make([]string, 0, len(fs))

	for _, f := range fs {
		res = append(res, filepath.Join(pkg.Dir, f))
	}

	return res
}

func isStdlib(pkg string) bool {
	return !strings.Contains(strings.SplitN(pkg, "/", 2)[0], ".")
}

// findImports recursively adds all imported packages by the given
// package (packageName) to the allPkgs map.
func findImports(ctx build.Context, packageName, dir string, recur bool, allPkgs map[string][]string, rootPkgs map[string]bool) error {
	if packageName == "C" {
		return nil
	}

	pkg, err := ctx.Import(packageName, dir, 0)
	if err != nil {
		return errors.Wrapf(err, "cannot find %q", packageName)
	}

	// Iterate through the imports in sorted order so that we provide
	// deterministic results.
	for _, name := range imports(pkg, rootPkgs[pkg.ImportPath]) {
		if isStdlib(name) {
			continue
		}

		_, alreadyDone := allPkgs[name]
		allPkgs[name] = append(allPkgs[name], pkg.ImportPath)
		if recur && !alreadyDone {
			if err := findImports(ctx, name, pkg.Dir, recur, allPkgs, rootPkgs); err != nil {
				return err
			}
		}
	}

	return nil
}

func imports(pkg *build.Package, isRoot bool) []string {
	var res []string

	for _, s := range pkg.Imports {
		if isStdlib(s) {
			continue
		}

		res = append(res, s)
	}

	return res
}
