package baur

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/pkg/errors"

	"github.com/simplesurance/baur/fs"
	"github.com/simplesurance/baur/golang"
)

// GoSrcDirs resolves Golang source files in directories to files including
// resolving all imports to files
type GoSrcDirs struct {
	repositoryRootPath string
	relAppPath         string
	paths              []string
	gopath             string
}

// NewGoSrcDirs returns a GoSrcDirs
func NewGoSrcDirs(repositoryRootPath, relAppPath, gopath string, paths []string) *GoSrcDirs {
	return &GoSrcDirs{
		repositoryRootPath: repositoryRootPath,
		relAppPath:         relAppPath,
		paths:              paths,
		gopath:             gopath,
	}
}

// Resolve returns list of Go src files
func (g *GoSrcDirs) Resolve() ([]BuildInput, error) {
	baseDir := filepath.Join(g.repositoryRootPath, g.relAppPath)
	fullpaths := make([]string, 0, len(g.paths))

	for _, p := range g.paths {
		absPath := filepath.Join(baseDir, p)

		isDir, err := fs.IsDir(absPath)
		if err != nil {
			return nil, err
		}

		if !isDir {
			return nil, fmt.Errorf("%q is not a directory", p)
		}

		fullpaths = append(fullpaths, absPath)
	}

	absSrcPaths, err := golang.SrcFiles(g.gopath, fullpaths...)
	if err != nil {
		return nil, err
	}

	res := make([]BuildInput, 0, len(absSrcPaths))
	for _, p := range absSrcPaths {
		relPath, err := filepath.Rel(g.repositoryRootPath, p)
		if err != nil {
			return nil, errors.Wrapf(err, "converting %q to relpath with basedir %q failed", p, g.repositoryRootPath)
		}

		res = append(res, NewFile(g.repositoryRootPath, relPath))
	}

	return res, nil
}

// Type returns the type of resolver
func (g *GoSrcDirs) Type() string {
	return "GolangSources"
}

// String returns the GoPath and Paths to resolve
func (g *GoSrcDirs) String() string {
	return fmt.Sprintf("GOPATH: \"%s\", Paths: \"%s\"", g.gopath, strings.Join(g.paths, ", "))
}
