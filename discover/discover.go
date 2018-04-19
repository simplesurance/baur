// package discover provides functionality to discover a repository and it's
// applications.
package discover

import (
	"os"
	"path"
	"path/filepath"

	"github.com/pkg/errors"
)

// RepositoryRoot searches for a file with the given name in the current
// directory and in it's sparent directories.
// If the file is found it returns the path to the directory
func RepositoryRoot(filename string) (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}

	for {
		p := path.Join(dir, filename)

		_, err = os.Stat(p)
		if err == nil {
			dir := path.Dir(filename)

			abs, err := filepath.Abs(dir)
			if err != nil {
				return "", errors.Wrapf(err,
					"could not get absolute path of %v", dir)
			}

			return abs, nil
		}

		if !os.IsNotExist(err) {
			return "", nil
		}

		// TODO: how to detect OS independent if reached the root dir
		if dir == "/" {
			return "", os.ErrNotExist
		}

		dir = path.Join(dir, "..")
	}
}

// ApplicationDirs returns all directories containing fname. The function
// searches through all basedirs up to lvl directories deep.
// If lvl is 0 only the basedir is searched.
func ApplicationDirs(basedirs []string, fname string, lvl int) ([]string, error) {
	var found []string

	for _, basedir := range basedirs {
		glob := "/"

		for i := 0; i <= lvl; i++ {
			p := path.Join(basedir, glob, fname)

			matches, err := filepath.Glob(p)
			if err != nil {
				return nil, err
			}

			for _, m := range matches {
				found = append(found, path.Dir(m))
			}

			glob += "*/"
		}
	}

	return found, nil
}
