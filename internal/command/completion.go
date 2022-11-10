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
	_ string,
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

type completeTargetFuncOpts struct {
	withoutWildcards bool
	withoutPaths     bool
	withoutAppNames  bool
}

func newCompleteTargetFunc(
	opts completeTargetFuncOpts,
) func(*cobra.Command, []string, string) ([]string, cobra.ShellCompDirective) {
	return func(_ *cobra.Command, _ []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		repo, err := findRepository()
		if err != nil {
			return nil, cobra.ShellCompDirectiveError
		}

		loader, err := baur.NewLoader(repo.Cfg, mockGitCommitID, log.StdLogger)
		if err != nil {
			return nil, cobra.ShellCompDirectiveError
		}

		tasks, err := loader.LoadTasks()
		if err != nil {
			return nil, cobra.ShellCompDirectiveError
		}

		wd := ""
		if !opts.withoutPaths {
			wd, _ = os.Getwd()
		}

		resultSet := make(map[string]struct{}, len(tasks)*3)
		for _, t := range tasks {
			resultSet[t.ID()] = struct{}{}
			if !opts.withoutWildcards {
				resultSet["*."+t.ID()] = struct{}{}
			}

			if !opts.withoutAppNames {
				if _, exist := resultSet[t.AppName]; exist {
					continue
				}
			}

			resultSet[t.AppName] = struct{}{}
			if !opts.withoutWildcards {
				resultSet[t.AppName+".*"] = struct{}{}
			}

			if wd != "" {
				appRelPath, err := filepath.Rel(wd, t.Directory)
				if err == nil {
					resultSet[appRelPath] = struct{}{}
				}
			}
		}

		result := make([]string, 0, len(resultSet))
		for k := range resultSet {
			result = append(result, k)
		}

		return result, cobra.ShellCompDirectiveDefault
	}
}

func completeOnlyDirectories(_ *cobra.Command, _ []string, _ string) ([]string, cobra.ShellCompDirective) {
	return nil, cobra.ShellCompDirectiveFilterDirs
}
