package command

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/simplesurance/baur/v3/internal/command/flag"
	"github.com/simplesurance/baur/v3/internal/command/term"
	"github.com/simplesurance/baur/v3/internal/format"
	"github.com/simplesurance/baur/v3/internal/format/csv"
	"github.com/simplesurance/baur/v3/internal/format/table"
	"github.com/simplesurance/baur/v3/internal/log"
	"github.com/simplesurance/baur/v3/pkg/baur"
	"github.com/simplesurance/baur/v3/pkg/storage"
)

const (
	statusAppNameHeader   = "Application Name"
	statusAppNameParam    = "app-name"
	statusTaskIDHeader    = "Task ID"
	statusTaskIDParam     = "task-id"
	statusPathHeader      = "Path"
	statusPathParam       = "path"
	statusStatusHeader    = "Status"
	statusStatusParam     = "status"
	statusRunIDHeader     = "Run ID"
	statusRunIDParam      = "run-id"
	statusGitCommitHeader = "Git Commit"
	statusGitCommitParam  = "git-commit"
)

var statusLongHelp = fmt.Sprintf(
	`List the task status in the repository.

Arguments:
%s`,
	targetHelp)

func init() {
	rootCmd.AddCommand(&newStatusCmd().Command)
}

type statusCmd struct {
	cobra.Command

	csv                     bool
	quiet                   bool
	absPaths                bool
	inputStr                []string
	lookupInputStr          string
	buildStatus             flag.TaskStatus
	fields                  *flag.Fields
	requireCleanGitWorktree bool
}

func newStatusCmd() *statusCmd {
	cmd := statusCmd{
		Command: cobra.Command{
			Use:               "status [TARGET|APP_DIR]...",
			Short:             "list status of tasks",
			Long:              statusLongHelp,
			Args:              cobra.ArbitraryArgs,
			ValidArgsFunction: newCompleteTargetFunc(completeTargetFuncOpts{}),
		},

		fields: flag.MustNewFields(
			[]string{
				statusAppNameParam,
				statusTaskIDParam,
				statusPathParam,
				statusRunIDParam,
				statusStatusParam,
				statusGitCommitParam,
			},
			[]string{
				statusTaskIDParam,
				statusPathParam,
				statusRunIDParam,
				statusStatusParam,
				statusGitCommitParam,
			},
		),
	}
	cmd.Run = cmd.run

	cmd.Flags().BoolVar(&cmd.csv, "csv", false,
		"List applications in RFC4180 CSV format")

	cmd.Flags().BoolVarP(&cmd.quiet, "quiet", "q", false,
		"Suppress printing a header and progress dots")

	cmd.Flags().BoolVar(&cmd.absPaths, "abs-path", false,
		"Show absolute instead of relative paths")

	cmd.Flags().VarP(&cmd.buildStatus, "status", "s",
		cmd.buildStatus.Usage(term.Highlight))

	cmd.Flags().VarP(cmd.fields, "fields", "f",
		cmd.fields.Usage(term.Highlight))

	cmd.Flags().StringArrayVar(&cmd.inputStr, "input-str", nil,
		"include a string as input, can be specified multiple times")

	cmd.Flags().StringVar(&cmd.lookupInputStr, "lookup-input-str", "",
		"if a run can not be found, try to find a run with this value as input-string")

	cmd.Flags().BoolVarP(&cmd.requireCleanGitWorktree, flagNameRequireCleanGitWorktree, "c", false,
		"fail if the git repository contains modified or untracked files")

	return &cmd
}

func (c *statusCmd) statusCreateHeader() []string {
	var headers []string

	for _, f := range c.fields.Fields {
		switch f {
		case statusAppNameParam:
			headers = append(headers, statusAppNameHeader)
		case statusTaskIDParam:
			headers = append(headers, statusTaskIDHeader)
		case statusPathParam:
			headers = append(headers, statusPathHeader)
		case statusStatusParam:
			headers = append(headers, statusStatusHeader)
		case statusRunIDParam:
			headers = append(headers, statusRunIDHeader)
		case statusGitCommitParam:
			headers = append(headers, statusGitCommitHeader)

		default:
			panic(fmt.Sprintf("unsupported value '%v' in fields parameter", f))
		}
	}

	return headers
}

func (c *statusCmd) run(_ *cobra.Command, args []string) {
	var headers []string
	var formatter format.Formatter
	var storageClt storage.Storer

	repo := mustFindRepository()
	vcsState := mustGetRepoState(repo.Path)

	mustUntrackedFilesNotExist(c.requireCleanGitWorktree, vcsState)

	loader, err := baur.NewLoader(
		repo.Cfg,
		vcsState.CommitID,
		log.StdLogger,
	)
	exitOnErr(err)

	tasks, err := loader.LoadTasks(args...)
	exitOnErr(err)

	writeHeaders := !c.quiet && !c.csv
	storageQueryNeeded := c.storageQueryIsNeeded()

	if storageQueryNeeded {
		storageClt = mustNewCompatibleStorage(repo)
		defer storageClt.Close()
	}

	if writeHeaders {
		headers = c.statusCreateHeader()
	}

	if c.csv {
		formatter = csv.New(headers, stdout)
	} else {
		formatter = table.New(headers, stdout)
	}

	showProgress := len(tasks) >= 5 && !c.quiet && !c.csv

	statusMgr := baur.NewTaskStatusEvaluator(repo.Path, storageClt, baur.NewInputResolver(mustGetRepoState(repo.Path)), c.inputStr, c.lookupInputStr)

	baur.SortTasksByID(tasks)

	for i, task := range tasks {
		var row []interface{}
		var taskRun *storage.TaskRunWithID
		var taskStatus baur.TaskStatus

		if storageQueryNeeded {
			var err error

			taskStatus, _, taskRun, err = statusMgr.Status(ctx, task)
			exitOnErrf(err, "%s: evaluating task status failed", task)

			// querying the build status for all applications can
			// take some time, output progress dots to let the user
			// know that something is happening
			if showProgress {
				stdout.Printf(".")

				if i+1 == len(tasks) {
					stdout.Printf("\n\n")
				}
			}
		}

		if c.buildStatus.IsSet() && taskStatus != c.buildStatus.Status {
			continue
		}

		row = c.statusAssembleRow(repo.Path, task, taskRun, taskStatus)

		mustWriteRow(formatter, row...)
	}

	err = formatter.Flush()
	exitOnErr(err)
}

func (c *statusCmd) storageQueryIsNeeded() bool {
	if c.buildStatus.IsSet() {
		return true
	}

	for _, f := range c.fields.Fields {
		switch f {
		case statusStatusParam:
			return true
		case statusRunIDParam:
			return true
		case statusGitCommitParam:
			return true
		}
	}

	return false
}

func (c *statusCmd) statusAssembleRow(repositoryDir string, task *baur.Task, taskRun *storage.TaskRunWithID, buildStatus baur.TaskStatus) []interface{} {
	var row []interface{}

	for _, f := range c.fields.Fields {
		switch f {
		case statusAppNameParam:
			row = append(row, task.AppName)

		case statusTaskIDParam:
			row = append(row, task.ID())

		case statusPathParam:
			if c.absPaths {
				row = append(row, task.Directory)
			} else {
				row = append(row, mustTaskRepoRelPath(repositoryDir, task))
			}

		case statusStatusParam:
			row = append(row, buildStatus)

		case statusRunIDParam:
			if buildStatus == baur.TaskStatusRunExist {
				row = append(row, fmt.Sprint(taskRun.ID))
			} else {
				// no build exist, we don't have a build id
				row = append(row, "")
			}

		case statusGitCommitParam:
			if buildStatus == baur.TaskStatusRunExist {
				row = append(row, fmt.Sprint(taskRun.VCSRevision))
			} else {
				row = append(row, "")
			}
		}
	}

	return row
}
