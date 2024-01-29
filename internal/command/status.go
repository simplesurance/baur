package command

import (
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/simplesurance/baur/v3/internal/command/flag"
	"github.com/simplesurance/baur/v3/internal/command/term"
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
	format                  *flag.Format
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

		format: flag.NewFormatFlag(),
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

	cmd.Flags().Var(cmd.format, "format", cmd.format.Usage(term.Highlight))
	_ = cmd.format.RegisterFlagCompletion(&cmd.Command)

	cmd.Flags().BoolVar(&cmd.csv, "csv", false,
		"Output status in RFC4180 CSV format")
	_ = cmd.Flags().MarkDeprecated("csv", "use --format=csv instead")

	cmd.MarkFlagsMutuallyExclusive("format", "csv")

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

	cmd.PreRun = func(*cobra.Command, []string) {
		if cmd.csv {
			cmd.format.Val = flag.FormatCSV
		}
	}

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

	writeHeaders := !c.quiet && c.format.Val == flag.FormatPlain
	storageQueryNeeded := c.storageQueryIsNeeded()

	if storageQueryNeeded {
		storageClt = mustNewCompatibleStorage(repo)
		defer storageClt.Close()
	}

	if writeHeaders {
		headers = c.statusCreateHeader()
	}

	formatter := mustNewFormatter(c.format.Val, headers)

	showProgress := len(tasks) >= 5 && !c.quiet && c.format.Val == flag.FormatPlain

	statusMgr := baur.NewTaskStatusEvaluator(
		repo.Path,
		storageClt,
		baur.NewInputResolver(mustGetRepoState(repo.Path), repo.Path, !c.requireCleanGitWorktree),
		c.inputStr, c.lookupInputStr,
	)

	baur.SortTasksByID(tasks)

	var rows []*statusRow
	for i, task := range tasks {
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

		row := c.assembleRow(repo.Path, task, taskRun, taskStatus)
		if c.format.Val == flag.FormatJSON {
			rows = append(rows, row)
		} else {
			mustWriteRow(formatter, row.asOrderedSlice(c.fields.Fields)...)
		}
	}

	if c.format.Val == flag.FormatJSON {
		c.mustStatusRowsToJSON(rows)
		return
	}

	exitOnErr(formatter.Flush())
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

func strPtr(s string) *string {
	return &s
}

func (c *statusCmd) mustStatusRowsToJSON(rows []*statusRow) {
	// We don't marshal statusRow directly because we need custom
	// behaviour to distinguish fields that should not be shown
	// (not passed via --fields) and fields that are undefined
	// (null). When marshalling rows fields that are not defined
	// via "--fields" would always be shown as null.
	res := make([]map[string]any, 0, len(rows))
	for _, r := range rows {
		res = append(res, r.asMap(c.fields.Fields))
	}
	enc := json.NewEncoder(stdout)
	enc.SetIndent("", "  ")
	exitOnErr(enc.Encode(res))
}

func (c *statusCmd) assembleRow(repositoryDir string, task *baur.Task, taskRun *storage.TaskRunWithID, buildStatus baur.TaskStatus) *statusRow {
	var row statusRow

	for _, f := range c.fields.Fields {
		switch f {
		case statusAppNameParam:
			row.AppName = &task.AppName

		case statusTaskIDParam:
			row.TaskID = strPtr(task.ID())

		case statusPathParam:
			if c.absPaths {
				row.Path = &task.Directory
			} else {
				row.Path = strPtr(mustTaskRepoRelPath(repositoryDir, task))
			}

		case statusStatusParam:
			row.Status = strPtr(buildStatus.String())

		case statusRunIDParam:
			if buildStatus == baur.TaskStatusRunExist {
				row.RunID = strPtr(fmt.Sprint(taskRun.ID))
			}

		case statusGitCommitParam:
			if buildStatus == baur.TaskStatusRunExist {
				row.GitCommit = strPtr(fmt.Sprint(taskRun.VCSRevision))
			}
		}

	}
	return &row
}

type statusRow struct {
	AppName   *string
	GitCommit *string
	Path      *string
	RunID     *string
	Status    *string
	TaskID    *string
}

func (r *statusRow) asMap(order []string) map[string]any {
	m := make(map[string]any, len(order))
	for _, f := range order {
		switch f {
		case statusAppNameParam:
			m["AppName"] = r.AppName
		case statusTaskIDParam:
			m["TaskID"] = r.TaskID
		case statusPathParam:
			m["Path"] = r.Path
		case statusStatusParam:
			m["Status"] = r.Status
		case statusRunIDParam:
			m["RunID"] = r.RunID
		case statusGitCommitParam:
			m["GitCommit"] = r.GitCommit
		}
	}

	return m
}

func (r *statusRow) asOrderedSlice(order []string) []any {
	result := make([]any, 0, len(order))
	for _, f := range order {
		switch f {
		case statusAppNameParam:
			result = sliceAppendNilAsEmpty(result, r.AppName)
		case statusTaskIDParam:
			result = sliceAppendNilAsEmpty(result, r.TaskID)
		case statusPathParam:
			result = sliceAppendNilAsEmpty(result, r.Path)
		case statusStatusParam:
			result = sliceAppendNilAsEmpty(result, r.Status)
		case statusRunIDParam:
			result = sliceAppendNilAsEmpty(result, r.RunID)
		case statusGitCommitParam:
			result = sliceAppendNilAsEmpty(result, r.GitCommit)
		}
	}

	return result
}
