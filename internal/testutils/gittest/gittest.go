package gittest

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/simplesurance/baur/v1/internal/exec"
)

// CommitFilesToGit adds and commit all files in directory (incl.
// subdirectories) to git
func CommitFilesToGit(t *testing.T, directory string) []string {
	var files []string

	t.Helper()

	err := filepath.Walk(directory, func(path string, fi os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if fi.IsDir() && fi.Name() == ".git" {
			return filepath.SkipDir
		}

		if !fi.IsDir() {
			files = append(files, path)
		}

		return nil
	})

	require.NoError(t, err)

	_, err = exec.Command("git", append([]string{"add"}, files...)...).
		Directory(directory).
		ExpectSuccess().
		Run()
	require.NoError(t, err)

	_, err = exec.Command("git", "commit", "-a", "-m", "baur commit").
		Directory(directory).
		ExpectSuccess().
		Run()
	require.NoError(t, err)

	return files
}

func Clone(t *testing.T, directory, gitURL, commit string) {
	_, err := exec.Command("git", "clone", gitURL, directory).ExpectSuccess().Run()
	require.NoError(t, err)

	if commit == "" {
		return
	}

	_, err = exec.Command("git", "checkout", commit).Directory(directory).ExpectSuccess().Run()
	require.NoError(t, err)
}

func CreateRepository(t *testing.T, directory string) {
	_, err := exec.Command("git", "init", ".").Directory(directory).ExpectSuccess().Run()
	require.NoError(t, err)
}
