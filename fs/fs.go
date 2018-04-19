package fs

import (
	"fmt"
	"os"
)

// DirsExist runs DirExists for multiple paths.
func DirsExist(paths []string) error {
	for _, path := range paths {
		err := DirExists(path)
		if err != nil {
			return err
		}
	}

	return nil
}

// DirExists returns nil if the path is a directory.
func DirExists(path string) error {
	fi, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("'%s' does not exist", path)
		}
		return err
	}

	if fi.IsDir() {
		return nil
	}

	return fmt.Errorf("'%s' is not a directory", path)
}
