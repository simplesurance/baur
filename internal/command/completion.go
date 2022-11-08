package command

import (
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/simplesurance/baur/v3/internal/log"
	"github.com/simplesurance/baur/v3/pkg/baur"
)

func mockGitCommitID() (string, error) {
	return "123", nil
}

func completeAppNameAndAppDir(
	_ *cobra.Command,
	_ []string,
	toComplete string,
) ([]string, cobra.ShellCompDirective) {
	repo, err := findRepository()
	if err != nil {
		return nil, cobra.ShellCompDirectiveError
	}

	loader, err := baur.NewLoader(repo.Cfg, mockGitCommitID, log.StdLogger)
	if err != nil {
		return nil, cobra.ShellCompDirectiveError
	}

	apps, err := loader.LoadApps()
	if err != nil {
		return nil, cobra.ShellCompDirectiveError
	}

	wd, _ := os.Getwd()

	result := make([]string, 0, len(apps)*2)
	for _, app := range apps {
		result = append(result, app.Name)
		if wd != "" {
			relPath, err := filepath.Rel(wd, app.Path)
			if err == nil {
				result = append(result, relPath)
			}
		}
	}

	return result, cobra.ShellCompDirectiveDefault
}
