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

	"github.com/simplesurance/baur/v2/internal/exec"
	"github.com/simplesurance/baur/v2/internal/fs"
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
// - .git/ does not exist and the "git" command is not in $PATH or "git
//   rev-parse --git-dir" exits with code 128
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

// splitArgs counts the number of characters in the elements of args.
// If more characters then maxArgStrLen accumulated, a slice with all the
// elements up to the current one is returned as the first return value.
// The second return value are the remaining elements in args.
// If less characters then maxArgStrLen are in args, then the unchanged args
// slice is returned as first element, the second slice is an empty slice.
func splitArgs(args []string, maxArgStrLen int) ([]string, []string) {
	var argSize int

	for i, arg := range args {
		argSize += len(arg)
		if argSize > maxArgStrLen {
			return args[:i+1], args[i+1:]
		}
	}

	return args, nil
}

// LsFiles runs git ls-files in dir, passes args as argument and returns a list
// of paths. All pathspecs are treated literally, globs are not resolved.
func LsFiles(dir string, pathspec ...string) ([]string, error) {
	// The maximum size of arguments for exec() on windows is 32767, on
	// linux it is 1/4 of the stack size which is much higher on most
	// systems, on macOs it seems to be 256kb
	// (https://go-review.googlesource.com/c/go/+/229317/3/src/cmd/go/internal/work/exec.go).
	// 2048 is subtracted as recommended at
	// https://www.in-ulm.de/~mascheck/various/argmax/ to have some
	// (little) space for env vars.
	// The size we use for args might be slightly higher because of the
	// args that are prepended in lsFiles(). This does not really matter,
	// cause the reserved space for env vars is only an estimate and much
	// higher then needed in most scenarios.
	const maxArgs = 32767 - 2048

	return lsFilesArgSpl(maxArgs, dir, pathspec)
}

// lsFilesArgSpl exists to be able to test the functionality easier with a
// smaller argSplitStrLen value.
func lsFilesArgSpl(argSplitStrLen int, dir string, pathspec []string) ([]string, error) {
	result := make([]string, 0, len(pathspec))

	for len(pathspec) > 0 {
		var paths []string

		paths, pathspec = splitArgs(pathspec, argSplitStrLen)

		paths, err := lsFiles(dir, paths)
		if err != nil {
			return nil, err
		}

		result = append(result, paths...)
	}

	return result, nil
}

func lsFiles(dir string, pathspec []string) ([]string, error) {
	args := append(
		[]string{
			"--noglob-pathspecs",
			"-c", "core.quotepath=off",
			"ls-files",
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

	out := res.StrOutput()
	if len(out) == 0 {
		return []string{}, nil
	}

	paths := strings.Split(out, "\n")

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
