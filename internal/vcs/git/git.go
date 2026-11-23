// Package git provides functionality to interact with a Git repository.
package git

import (
	"bytes"
	"context"
	"errors"
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
// or file named ".git". In git worktrees and submodules ".git" is a file.
func IsGitDir(dir string) (bool, error) {
	for _, find := range []func(string, string) (string, error){fs.FindDirInParentDirs, fs.FindFileInParentDirs} {
		_, err := find(dir, ".git")
		if err == nil {
			return true, nil
		}

		if !errors.Is(err, os.ErrNotExist) {
			return false, err
		}
	}

	return false, nil
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
	const untrackedFileStatus = "??"
	const ignoredFileStatus = "!!"

	var res []string
	var stdout bytes.Buffer

	// -z: records are terminated by NUL and paths are not quoted, without
	// it git quotes paths containing e.g. spaces or non-ASCII characters.
	// Only stdout is parsed, warnings on stderr must not be part of a record.
	_, err := exec.
		Command("git", "status", "--porcelain", "-z", "--untracked-files=all", "--ignored").
		Stdout(&stdout).
		Directory(dir).ExpectSuccess().Run(context.TODO())
	if err != nil {
		return nil, err
	}

	records := strings.Split(stdout.String(), "\x00")
	for i := 0; i < len(records); i++ {
		// a record has the format "XY PATH"
		rec := records[i]
		if len(rec) < len("XY P") {
			continue
		}

		status, relPath := rec[:2], rec[3:]

		switch {
		case status == untrackedFileStatus || status == ignoredFileStatus:
			// on Windows git prints paths with forward slashes as
			// separator, convert them to windows backslash seperators via
			// filepath.FromSlash()
			res = append(res, filepath.FromSlash(relPath))

		case strings.ContainsAny(status, "RC"):
			// renamed and copied entries are followed by a record
			// containing the original path
			i++
		}
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
