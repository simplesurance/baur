package command

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/simplesurance/baur"
	"github.com/simplesurance/baur/format"
	"github.com/simplesurance/baur/format/csv"
	"github.com/simplesurance/baur/format/table"
	"github.com/simplesurance/baur/git"
	"github.com/simplesurance/baur/internal/command/flag"
	"github.com/simplesurance/baur/internal/command/terminal"
	"github.com/simplesurance/baur/log"
	"github.com/simplesurance/baur/storage"
)

const (
	statusNameHeader      = "Task ID"
	statusNameParam       = "task-id"
	statusPathHeader      = "Path"
	statusPathParam       = "path"
	statusStatusHeader    = "Status"
	statusStatusParam     = "status"
	statusRunIDHeader     = "Run ID"
	statusRunIDParam      = "run-id"
	statusGitCommitHeader = "Git Commit"
	statusGitCommitParam  = "git-commit"
)

func init() {
	rootCmd.AddCommand(&newStatusCmd().Command)
}

type statusCmd struct {
	cobra.Command

	csv         bool
	quiet       bool
	absPaths    bool
	buildStatus flag.TaskStatus
	fields      *flag.Fields
}

func newStatusCmd() *statusCmd {
	cmd := statusCmd{
		Command: cobra.Command{
			Use:   "status [<SPEC>|<PATH>]...",
			Short: "list status of tasks",
			Args:  cobra.ArbitraryArgs,
		},

		fields: flag.NewFields([]string{
			statusNameParam,
			statusPathParam,
			statusRunIDParam,
			statusStatusParam,
			statusGitCommitParam,
		}),
	}
	cmd.Run = cmd.run

	cmd.Flags().BoolVar(&cmd.csv, "csv", false,
		"List applications in RFC4180 CSV format")

	cmd.Flags().BoolVarP(&cmd.quiet, "quiet", "q", false,
		"Suppress printing a header and progress dots")

	cmd.Flags().BoolVar(&cmd.absPaths, "abs-path", false,
		"Show absolute instead of relative paths")

	// TODO: refactor buildStatus struct
	cmd.Flags().VarP(&cmd.buildStatus, "status", "s",
		cmd.buildStatus.Usage(terminal.Highlight))

	cmd.Flags().VarP(cmd.fields, "fields", "f",
		cmd.fields.Usage(terminal.Highlight))

	return &cmd
}

func (c *statusCmd) statusCreateHeader() []string {
	var headers []string

	for _, f := range c.fields.Fields {
		switch f {
		case statusNameParam:
			headers = append(headers, statusNameHeader)
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

func (c *statusCmd) run(cmd *cobra.Command, args []string) {
	var headers []string
	var formatter format.Formatter
	var storageClt storage.Storer

	repo := MustFindRepository()

	loader, err := baur.NewLoader(
		repo.Cfg,
		git.NewRepositoryState(repo.Path).CommitID,
		log.StdLogger,
	)
	exitOnErr(err)

	tasks, err := loader.LoadTasks(args...)
	exitOnErr(err)

	writeHeaders := !c.quiet && !c.csv
	storageQueryNeeded := c.storageQueryIsNeeded()

	if storageQueryNeeded {
		storageClt = mustNewCompatibleStorage(repo)
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

	statusMgr := baur.NewTaskStatusEvaluator(repo.Path, storageClt, baur.NewInputResolver())

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

		err := formatter.WriteRow(row)
		exitOnErr(err)
	}

	err = formatter.Flush()
	exitOnErr(err)
}

func (c *statusCmd) storageQueryIsNeeded() bool {
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
		case statusNameParam:
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
