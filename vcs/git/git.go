package git

import (
	"bufio"
	"bytes"
	"fmt"
	"os"
	stdexec "os/exec"
	"path"
	"regexp"
	"strings"

	"github.com/pkg/errors"

	"github.com/simplesurance/baur/v1/exec"
	"github.com/simplesurance/baur/v1/fs"
)

var gitLsPathSpecErrRe = regexp.MustCompile(`pathspec ('.+') did not match any file\(s\) known to git`)

func CommandIsInstalled() bool {
	_, err := stdexec.LookPath("git")

	return err == nil
}

// IsGitDir checks if the passed directory is in a git repository.
// It returns true if:
// - .git/ exists or
// - the "git" command is in $PATH and "git rev-parse --git-dir" returns exit code 0
// It returns false if:
// - .git/ does not exist and
// - the "git" command is not in $PATH or "git rev-parse --git-dir" exits with code 128

// If '.git/' exist, if it does not
func IsGitDir(dir string) (bool, error) {
	err := fs.DirsExist(path.Join(dir, ".git"))
	if err == nil {
		return true, nil
	}

	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return false, nil
		}

		return false, err
	}

	result, err := exec.Command("git", "rev-parse", "--git-dir").Directory(dir).Run()
	if err != nil {
		return false, err
	}

	if result.ExitCode == 0 {
		return true, nil
	}

	if result.ExitCode == 128 {
		return false, nil
	}

	return false, fmt.Errorf("executing %q in %q exited with code $d, expeted 0 or 128",
		result.Command, result.ExitCode)
}

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

// WorktreeIsDirty returns true if the repository contains modified files,
// untracked files are considered, files in .gitignore are ignored
func WorktreeIsDirty(dir string) (bool, error) {
	res, err := exec.Command("git", "status", "-s").Directory(dir).ExpectSuccess().Run()
	if err != nil {
		return false, err
	}

	if len(res.Output) == 0 {
		return false, nil
	}

	return true, nil
}
