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

	"github.com/simplesurance/baur/v5/internal/exec"
	"github.com/simplesurance/baur/v5/internal/fs"
)

// CommandIsInstalled returns true if an executable called "git" is found in
// the directories listed in the PATH environment variable.
func CommandIsInstalled() bool {
	_, err := stdexec.LookPath("git")

	return err == nil
}

// IsGitDir checks if the passed directory is part of a git repository.
// It returns true if dir or any of its parent directory containing a directory
// named ".git".
func IsGitDir(dir string) (bool, error) {
	_, err := fs.FindDirInParentDirs(dir, ".git")
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return false, nil
		}
		return false, err
	}

	return true, nil
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

// UntrackedFiles returns a list of untracked and modified files in the git repository.
// Files that exist and are in a .gitignore file are included.
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

// ObjectID calculates the git ID (hash) of a fille.
func ObjectID(ctx context.Context, absFilePath, repoRelFilePath string) (string, error) {
	// TODO: calculate the object ID instead of running an external command
	// git hash-object, or by calculating the ID on our own as described in https://git-scm.com/book/en/v2/Git-Internals-Git-Objects
	result, err := exec.Command("git", "hash-object", "--path", repoRelFilePath, absFilePath).
		ExpectSuccess().
		RunCombinedOut(ctx)
	if err != nil {
		return "", err
	}

	objectID := strings.TrimSpace(result.StrOutput())
	if objectID == "" {
		return "", errors.New("git returned nothing")
	}

	return objectID, nil
}
