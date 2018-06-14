package git

import (
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
