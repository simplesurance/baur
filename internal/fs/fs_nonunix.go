//go:build !unix

package fs

import (
	"errors"
	"io/fs"
)

// FileHasOwnerExecPerm always returns false, errors.ErrUnsupported.
func FileHasOwnerExecPerm(p string) (bool, error) {
	return false, errors.ErrUnsupported
}

// OwnerHasExecPerm always returns false.
func OwnerHasExecPerm(m fs.FileMode) bool {
	return false
}
