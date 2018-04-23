// package discover provides functionality to discover a repository and it's
// applications.
package discover

import (
	"os"
	"path"
	"path/filepath"

	"github.com/pkg/errors"
	"github.com/simplesurance/baur"
)

type Discover struct {
	appCfgReader baur.AppCfgReader
}

// New returns a new Discover object
func New(appCfgReader baur.AppCfgReader) *Discover {
	return &Discover{appCfgReader: appCfgReader}
}

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

// Applications finds applications in the repository.
// It searches for directories containing a file named fname. The function
// searches through all basedirs up to lvl directories deep.
// If lvl is 0 only the basedir is searched.
// When it finds the file it reads and parses the configuration and adds it to
// the result slice.
func (d *Discover) Applications(basedirs []string, fname string, lvl int) ([]*baur.App, error) {
	var found []*baur.App

	for _, basedir := range basedirs {
		glob := "/"

		for i := 0; i <= lvl; i++ {
			p := path.Join(basedir, glob, fname)

			matches, err := filepath.Glob(p)
			if err != nil {
				return nil, err
			}

			for _, m := range matches {
				appCfg, err := d.appCfgReader.AppFromFile(m)
				if err != nil {
					return nil, errors.Wrapf(err, "reading application config '%s' failed")
				}
				found = append(found,
					&baur.App{
						Name: appCfg.GetName(),
						Dir:  path.Dir(m),
					})
			}

			glob += "*/"
		}
	}

	return found, nil
}
