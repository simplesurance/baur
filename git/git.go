package git

import (
	"bufio"
	"fmt"
	"regexp"
	"strings"

	"github.com/pkg/errors"

	"github.com/simplesurance/baur/exec"
)

var gitLsPathSpecErrRe = regexp.MustCompile(`pathspec ('.+') did not match any file\(s\) known to git`)

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
// If no files match, ErrNotExist is returned
func LsFiles(cwd, args string) (string, error) {
	cmd := "git ls-files --error-unmatch " + args

	out, exitCode, err := exec.Command(cwd, cmd)
	if err != nil {
		return "", errors.Wrapf(err, "executing %q failed", cmd)
	}

	if exitCode != 0 {
		var errMsgs []string

		scanner := bufio.NewScanner(strings.NewReader(out))
		for scanner.Scan() {
			matches := gitLsPathSpecErrRe.FindStringSubmatch(scanner.Text())
			if len(matches) == 0 {
				continue
			}

			errMsgs = append(errMsgs, matches[1:]...)
		}

		if err := scanner.Err(); err != nil {
			return "", errors.Wrap(err, "scanning cmd output failed")
		}

		if len(errMsgs) != 0 {
			return "", errors.New("the following paths did not match any files: " + strings.Join(errMsgs, ", "))
		}

		return "", fmt.Errorf("%q exited with code %d, output: %q", cmd, exitCode, out)
	}

	return out, nil
}

// WorkTreeIsDirty returns true if the repository contains modified files,
// untracked files are considered, files in .gitignore are ignored
func WorkTreeIsDirty(dir string) (bool, error) {
	const cmd = "git status -s"

	out, exitCode, err := exec.Command(dir, cmd)
	if err != nil {
		return false, errors.Wrapf(err, "executing %q failed", cmd)
	}

	if exitCode != 0 {
		return false, fmt.Errorf("%q exited with code %d, output: %q", cmd, exitCode, out)
	}

	if len(out) == 0 {
		return false, nil
	}

	return true, nil
}
