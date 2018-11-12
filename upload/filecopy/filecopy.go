package filecopy

import (
	"io"
	"os"
	"path"

	"github.com/pkg/errors"

	"github.com/simplesurance/baur/fs"
)

var defLogFn = func(string, ...interface{}) { return }

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
		return errors.Wrapf(err, "opening %s failed", src)
	}

	srcFi, err := os.Stat(src)
	if err != nil {
		return errors.Wrapf(err, "stat %s failed", src)
	}

	srcFileMode := srcFi.Mode().Perm()

	dstFd, err := os.OpenFile(dst, os.O_RDWR|os.O_CREATE|os.O_TRUNC, srcFileMode)
	if err != nil {
		srcFd.Close()
		return errors.Wrapf(err, "opening %s failed", dst)
	}

	_, err = io.Copy(dstFd, srcFd)
	if err != nil {
		return err
	}

	if err = srcFd.Close(); err != nil {
		return err
	}

	if err = dstFd.Close(); err != nil {
		return err
	}

	return err
}

// Upload copies the file with src path to the dst path.
// If the destination directory does not exist, it is created.
// If the destination path exist and is not a regular file an error is returned.
// If it exist and is a file, the file is overwritten if it's not the same.
func (c *Client) Upload(src string, dst string) (string, error) {
	destDir := path.Dir(dst)

	isDir, err := fs.IsDir(destDir)
	if err != nil {
		if !os.IsNotExist(err) {
			return "", err
		}

		err = fs.Mkdir(destDir)
		if err != nil {
			return "", errors.Wrapf(err, "creating directory '%s' failed", destDir)
		}

		c.debugLogFn("filecopy: created directory '%s'", destDir)
	} else {
		if !isDir {
			return "", errors.Wrapf(err, "%s is not a directory", destDir)
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
		return "", errors.Wrapf(err, "'%s' exist but is not a regular file", dst)
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
