package git

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"os"
	stdexec "os/exec"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/simplesurance/baur/v1/internal/exec"
	"github.com/simplesurance/baur/v1/internal/fs"
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
	err := fs.DirsExist(filepath.Join(dir, ".git"))
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

	return false, fmt.Errorf("executing %q in %q exited with code $d, expected 0 or 128",
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
		return "", errors.New("executing git rev-parse HEAD failed, no Stdout output")
	}

	return commitID, err
}

// LsFiles runs git ls-files in dir, passes args as argument and returns a list
// of paths . If a patchspec matches no files ErrNotExist is returned.
// All pathspecs are treated literally, globs are not resolved.
func LsFiles(dir string, pathspec ...string) ([]string, error) {
	args := append(
		[]string{
			"--noglob-pathspecs",
			"-c", "core.quotepath=off",
			"ls-files",
			"--error-unmatch",
		},
		pathspec...)

	res, err := exec.Command("git", args...).Directory(dir).Run()
	if err != nil {
		return nil, err
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
			return nil, fmt.Errorf("scanning cmd output failed: %w", err)
		}

		if len(errMsgs) != 0 {
			return nil, errors.New("the following paths did not match any files: " + strings.Join(errMsgs, ", "))
		}

		return nil, res.ExpectSuccess()
	}

	paths := strings.Split(res.StrOutput(), "\n")

	return paths, nil
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
