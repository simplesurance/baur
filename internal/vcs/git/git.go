// Package git provides functionality to interact with a Git repository.
package git

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	stdexec "os/exec"
	"path/filepath"
	"strings"

	"github.com/simplesurance/baur/v3/internal/exec"
	"github.com/simplesurance/baur/v3/internal/fs"
)

// CommandIsInstalled returns true if an executable called "git" is found in
// the directories listed in the PATH environment variable.
func CommandIsInstalled() bool {
	_, err := stdexec.LookPath("git")

	return err == nil
}

// IsGitDir checks if the passed directory is in a git repository.
// It returns true if:
// - .git/ exists or
// - the "git" command is in $PATH and "git rev-parse --git-dir" returns exit code 0
// It returns false if:
//   - .git/ does not exist and the "git" command is not in $PATH or "git
//     rev-parse --git-dir" exits with code 128
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

	result, err := exec.Command("git", "rev-parse", "--git-dir").Directory(dir).RunCombinedOut(context.TODO())
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
	res, err := exec.Command("git", "rev-parse", "HEAD").Directory(dir).ExpectSuccess().RunCombinedOut(context.TODO())
	if err != nil {
		return "", err
	}

	commitID := strings.TrimSpace(res.StrOutput())
	if len(commitID) == 0 {
		return "", errors.New("executing git rev-parse HEAD failed, no Stdout output")
	}

	return commitID, err
}

// WorktreeIsDirty returns true if the repository contains modified files,
// untracked files are considered, files in .gitignore are ignored
func WorktreeIsDirty(dir string) (bool, error) {
	res, err := exec.Command("git", "status", "-s").Directory(dir).ExpectSuccess().RunCombinedOut(context.TODO())
	if err != nil {
		return false, err
	}

	if len(res.CombinedOutput) == 0 {
		return false, nil
	}

	return true, nil
}

// UntrackedFiles returns a list of untracked files in the repository found at dir.
// The returned paths are relative to dir.
func UntrackedFiles(dir string) ([]string, error) {
	const untrackedFilePrefix = "?? "
	const ignoredFilePrefix = "!! "

	var res []string

	cmdResult, err := exec.
		Command("git", "status", "--porcelain", "--untracked-files=all", "--ignored").
		Directory(dir).ExpectSuccess().RunCombinedOut((context.TODO()))
	if err != nil {
		return nil, err
	}

	scanner := bufio.NewScanner(bytes.NewReader(cmdResult.CombinedOutput))
	for scanner.Scan() {
		var relPath string

		line := scanner.Text()
		//nolint:gocritic // ifElseChain: rewrite if-else to switch statement
		if strings.HasPrefix(line, untrackedFilePrefix) {
			relPath = strings.TrimPrefix(line, untrackedFilePrefix)
		} else if strings.HasPrefix(line, ignoredFilePrefix) {
			relPath = strings.TrimPrefix(line, ignoredFilePrefix)
		} else {
			continue
		}

		// on Windows git prints paths with forward slashes as
		// separator, convert them to windows backslash seperators via
		// filepath.FromSlash()
		res = append(res, filepath.FromSlash(relPath))
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("scanning git status output failed: %w", err)
	}

	return res, nil
}
