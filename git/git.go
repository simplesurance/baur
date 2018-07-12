package git

import (
	"fmt"
	"strings"

	"github.com/pkg/errors"

	"github.com/simplesurance/baur/exec"
)

// CommitID return the commit id of HEAD by running git rev-parse in the passed
// directory
func CommitID(dir string) (string, error) {
	out, exitCode, err := exec.Command(dir, "git rev-parse HEAD")
	if err != nil {
		return "", errors.Wrap(err, "executing git rev-parse HEAD failed")
	}

	if exitCode != 0 {
		return "", errors.Wrapf(err, "executing git rev-parse HEAD failed, output: %q", out)
	}

	commitID := strings.TrimSpace(out)
	if len(commitID) == 0 {
		return "", errors.Wrap(err, "executing git rev-parse HEAD failed, no Stdout output")
	}

	return commitID, err
}

// LsFiles runs git ls-files in dir, passes args as argument and returns the
// output
func LsFiles(dir, args string) (string, error) {
	cmd := "git ls-files " + args

	out, exitCode, err := exec.Command(dir, cmd)
	if err != nil {
		return "", errors.Wrapf(err, "executing %q failed", cmd)
	}

	if exitCode != 0 {
		return "", fmt.Errorf("%q exited with code %d, output: %q", cmd, exitCode, out)
	}

	return out, nil
}
