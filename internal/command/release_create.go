package command

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/simplesurance/baur/v3/internal/command/term"
	"github.com/simplesurance/baur/v3/internal/log"
	"github.com/simplesurance/baur/v3/pkg/baur"
	"github.com/simplesurance/baur/v3/pkg/storage"

	"github.com/spf13/cobra"
)

const releaseCreateExample = `
baur release create stable		      create a release named 'stable',
					      including all tasks
baur release create --include '*.build' v3    create a release named 'v3',
					      including only tasks called
					      'build'`

var releaseCreateLongHelp = fmt.Sprintf(`
Creates a named snapshot of the status of tasks.
The snapshot includes information about the tasks and all their outputs.

All tasks included in the release must have a run state of Exist.
Additional information can be stored with the release database.
Because it is stored in database, this should not consume too much disk space.

By default all tasks are included in the release. This can be narrowed via the
--include flag. See %s for more information about the TARGET syntax.

Exit Codes:
  0 - Success
  1 - Error
  %d - Release with the same name already exist
  %d - One or more tasks are in pending state
`,
	term.Highlight("baur run --help"),
	exitCodeAlreadyExist, exitCodeTaskRunIsPending)

type releaseCreateCmd struct {
	cobra.Command
	requireCleanGitWorktree bool

	metadataFile string
	includes     []string
}

func init() {
	releaseCmd.AddCommand(&newReleaseCreateCmd().Command)
}

func newReleaseCreateCmd() *releaseCreateCmd {
	cmd := releaseCreateCmd{
		Command: cobra.Command{
			Use:               "create NAME",
			Short:             "create a release",
			Long:              strings.TrimSpace(releaseCreateLongHelp),
			Args:              cobra.ExactArgs(1),
			Example:           strings.TrimSpace(releaseCreateExample),
			ValidArgsFunction: nil, // FIXME: implement completion
		},
	}

	cmd.Flags().BoolVarP(&cmd.requireCleanGitWorktree, flagNameRequireCleanGitWorktree, "c", false,
		"fail if the git repository contains modified or untracked files")
	cmd.Flags().StringVarP(&cmd.metadataFile, "metadata", "m", "",
		"path to a file containing additional data that is stored with the release",
	)
	cmd.Flags().StringSliceVar(&cmd.includes, "include", nil,
		"tasks to include in the release, by default all are included,\n"+
			"supports the same TARGET syntax then 'baur run'")

	cmd.Run = cmd.run

	return &cmd
}

func (c *releaseCreateCmd) run(cmd *cobra.Command, args []string) {
	ctx := cmd.Context()
	releaseName := args[0]

	stdout.Printf("creating release: %s\n", term.Highlight(releaseName))

	repo := mustFindRepository()
	vcsState := mustGetRepoState(repo.Path)

	mustUntrackedFilesNotExist(c.requireCleanGitWorktree, vcsState)

	loader, err := baur.NewLoader(
		repo.Cfg,
		vcsState.CommitID,
		log.StdLogger,
	)
	exitOnErr(err)

	stdout.Println("loading application configs...")
	tasks, err := loader.LoadTasks(c.includes...)
	exitOnErr(err)

	if len(tasks) == 0 {
		if len(c.includes) != 0 {
			fatalf("could not find any tasks matching %s", strings.Join(c.includes, ","))
		}
		fatalf("could not find any tasks in the baur repository")
	}

	storageClt := mustNewCompatibleStorageRepo(repo)
	runIDs := c.fetchTaskIDs(ctx, repo, storageClt, tasks)

	release := storage.Release{Name: releaseName, TaskRunIDs: runIDs}
	if c.metadataFile != "" {
		var err error
		metadataReader, err := os.Open(c.metadataFile)
		exitOnErrf(err, "opening metadata file failed")

		defer metadataReader.Close()

		release.Metadata = metadataReader
	}

	err = storageClt.CreateRelease(ctx, &release)
	if errors.Is(err, storage.ErrExists) {
		stderr.PrintErrf("release with name %q already exist, release names must be unique", releaseName)
		exitFunc(exitCodeAlreadyExist)
	}
	exitOnErr(err, "storing release information in database failed")

	stdout.Printf(
		"release %s created %s\n",
		term.Highlight(releaseName),
		term.GreenHighlight("successfully"),
	)

}

func (c *releaseCreateCmd) fetchTaskIDs(
	ctx context.Context,
	repo *baur.Repository,
	storageClt storage.Storer,
	tasks []*baur.Task,
) []int {
	statusMgr := baur.NewTaskStatusEvaluator(
		repo.Path,
		storageClt,
		baur.NewInputResolver(
			mustGetRepoState(repo.Path),
			repo.Path,
			nil,
			!c.requireCleanGitWorktree,
		),
		"",
	)

	runIDs := make([]int, 0, len(tasks))
	stdout.Printf("evaluating task statuses")
	for _, task := range tasks {
		status, _, taskRun, err := statusMgr.Status(ctx, task)
		if err != nil {
			stdout.Println("")
			exitOnErrf(err, "%s: evaluating task status failed", task)
		}
		if status != baur.TaskStatusRunExist {
			stdout.Println("")
			stderr.PrintErrf("%s: task status is %s, expecting %s\n",
				task.ID, status, baur.TaskStatusRunExist)
			exitFunc(exitCodeTaskRunIsPending)
		}

		stdout.Printf(".")
		runIDs = append(runIDs, taskRun.ID)

	}
	stdout.Println("")

	return runIDs
}
