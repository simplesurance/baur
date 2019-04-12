package git

import (
	"bufio"
	"bytes"
	"regexp"
	"strings"

	"github.com/pkg/errors"

	"github.com/simplesurance/baur/exec"
)

var gitLsPathSpecErrRe = regexp.MustCompile(`pathspec ('.+') did not match any file\(s\) known to git`)

// CommitID return the commit id of HEAD by running git rev-parse in the passed
// directory
func CommitID(dir string) (string, error) {
	res, err := exec.Command("git", "rev-parse", "HEAD").Directory(dir).ExpectSuccess().Run()
	if err != nil {
		return "", err
	}

	commitID := strings.TrimSpace(res.StrOutput())
	if len(commitID) == 0 {
		return "", errors.Wrap(err, "executing git rev-parse HEAD failed, no Stdout output")
	}

	return commitID, err
}

// LsFiles runs git ls-files in dir, passes args as argument and returns the
// output
// If no files match, ErrNotExist is returned
func LsFiles(dir string, arg ...string) (string, error) {
	args := append([]string{"-c", "core.quotepath=off", "ls-files", "error-unmatch"}, arg...)

	res, err := exec.Command("git", args...).Directory(dir).Run()
	if err != nil {
		return "", err
	}

	if res.ExitCode != 0 {
		var errMsgs []string

		scanner := bufio.NewScanner(bytes.NewReader(res.Output))
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

		return "", res.ExpectSuccess()
	}

	return res.StrOutput(), nil
}

// WorkTreeIsDirty returns true if the repository contains modified files,
// untracked files are considered, files in .gitignore are ignored
func WorkTreeIsDirty(dir string) (bool, error) {
	const cmd = "git status -s"

	res, err := exec.Command("git", "status", "-s").Directory(dir).ExpectSuccess().Run()
	if err != nil {
		return false, err
	}

	if len(res.Output) == 0 {
		return false, nil
	}

	return true, nil
}
