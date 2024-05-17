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

const (
	exitCodeAlreadyExist     = 2
	exitCodeTaskRunIsPending = 3
)

const createReleaseExample = `
baur create release stable		      create a release named 'stable',
					      including the state of all tasks
					      in the repository
baur create release --include '*.build' v3    create a release named 'v3',
					      including only all tasks called
					      build' `

var createReleaseLongHelp = fmt.Sprintf(`
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

type createReleaseCmd struct {
	cobra.Command
	requireCleanGitWorktree bool

	metadataFile string
	targets      []string
}

func init() {
	createCmd.AddCommand(&newCreateReleaseCmd().Command)
}

func newCreateReleaseCmd() *createReleaseCmd {
	cmd := createReleaseCmd{
		Command: cobra.Command{
			Use:               "release NAME",
			Short:             "create a release",
			Long:              strings.TrimSpace(createReleaseLongHelp),
			Args:              cobra.ExactArgs(1),
			Example:           strings.TrimSpace(createReleaseExample),
			ValidArgsFunction: nil, // TODO: implement completion
		},
	}

	cmd.Flags().BoolVarP(&cmd.requireCleanGitWorktree, flagNameRequireCleanGitWorktree, "c", false,
		"fail if the git repository contains modified or untracked files")
	cmd.Flags().StringVarP(&cmd.metadataFile, "file", "f", "",
		"path to a file containing additional data that is stored with the release.",
	)
	cmd.Flags().StringSliceVar(&cmd.targets, "include", []string{"*"},
		"tasks to include in the release, by default all are included,\n"+
			"supports the same TARGET syntax then 'baur run'")

	cmd.Run = cmd.run

	return &cmd
}

func (c *createReleaseCmd) run(cmd *cobra.Command, args []string) {
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
	tasks, err := loader.LoadTasks(args[1:]...)
	exitOnErr(err)

	storageClt := mustNewCompatibleStorage(repo)
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

func (c *createReleaseCmd) fetchTaskIDs(
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
