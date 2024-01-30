//go:build unix

package fs

import (
	"io/fs"
	"os"
)

// FileHasOwnerExecPerm returns true if the executable mode bit for the file
// owner is set.
func FileHasOwnerExecPerm(p string) (bool, error) {
	fi, err := os.Stat(p)
	if err != nil {
		return false, err
	}

	return OwnerHasExecPerm(fi.Mode()), nil
}

// OwnerHasExecPerm returns true if the executable mode bit in m is set.
func OwnerHasExecPerm(m fs.FileMode) bool {
	return m&0100 == 0100
}
