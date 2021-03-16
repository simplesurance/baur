package filecopy

import (
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/simplesurance/baur/v1/internal/fs"
)

var defLogFn = func(string, ...interface{}) {}

// Client copies files from one path to another
type Client struct {
	debugLogFn func(string, ...interface{})
}

// New returns a client
func New(debugLogFn func(string, ...interface{})) *Client {
	logFn := defLogFn
	if debugLogFn != nil {
		logFn = debugLogFn
	}

	return &Client{debugLogFn: logFn}
}

func copyFile(src, dst string) error {
	srcFd, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("opening %s failed: %w", src, err)
	}

	// nolint: errcheck
	defer srcFd.Close()

	srcFi, err := os.Stat(src)
	if err != nil {
		return fmt.Errorf("stat %s failed: %w", src, err)
	}

	srcFileMode := srcFi.Mode().Perm()

	dstFd, err := os.OpenFile(dst, os.O_RDWR|os.O_CREATE|os.O_TRUNC, srcFileMode)
	if err != nil {
		return fmt.Errorf("opening %s failed: %w", dst, err)
	}

	_, err = io.Copy(dstFd, srcFd)
	if err != nil {
		_ = dstFd.Close()

		return err
	}

	return dstFd.Close()
}

// Upload copies the file with src path to the dst path.
// If the destination directory does not exist, it is created.
// If the destination path exist and is not a regular file an error is returned.
// If it exists, is a file and it differs the file is overwritten.
func (c *Client) Upload(src string, dst string) (string, error) {
	destDir := filepath.Dir(dst)

	isDir, err := fs.IsDir(destDir)
	if err != nil {
		if !os.IsNotExist(err) {
			return "", err
		}

		err = fs.Mkdir(destDir)
		if err != nil {
			return "", fmt.Errorf("creating directory %s failed: %w", destDir, err)
		}
		c.debugLogFn("filecopy: created directory '%s'", destDir)
	} else {
		if !isDir {
			return "", fmt.Errorf("%s is not a directory", destDir)
		}
	}

	regFile, err := fs.IsRegularFile(dst)
	if err != nil {
		if !os.IsNotExist(err) {
			return "", err
		}

		return dst, copyFile(src, dst)
	}

	if !regFile {
		return "", fmt.Errorf("%s exist but is not a regular file", dst)
	}

	sameFile, err := fs.SameFile(src, dst)
	if err != nil {
		return "", err
	}

	if sameFile {
		c.debugLogFn("filecopy: '%s' already exist and is the same then '%s'", dst, src)
		return dst, nil
	}

	c.debugLogFn("filecopy: '%s' already exist, overwriting file", dst)

	return dst, copyFile(src, dst)
}
