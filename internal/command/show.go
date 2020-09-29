package command

import (
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/spf13/cobra"

	"github.com/simplesurance/baur/v1"
	"github.com/simplesurance/baur/v1/internal/command/term"
	"github.com/simplesurance/baur/v1/internal/format"
	"github.com/simplesurance/baur/v1/internal/format/table"
	"github.com/simplesurance/baur/v1/internal/log"
	"github.com/simplesurance/baur/v1/storage"
)

const showLongHelp = `
Show information about an application or task run.

If the name or path of an application directory is passed,
application information are shown.
If a numeric task-run ID is passed, information about the
recorded task run are shown.
`

const showExamples = `
baur show calc		show information about the calc application
baur show ui/shop	show information about the app in the ui/shop directory
baur show 512		show information about build 512
`

func init() {
	rootCmd.AddCommand(&newShowCmd().Command)
}

type showCmd struct {
	cobra.Command
}

func newShowCmd() *showCmd {
	cmd := showCmd{
		Command: cobra.Command{
			Use:     "show APP|APP-PATH|TASK-RUN-ID",
			Short:   "show information about apps or recorded task runs",
			Args:    cobra.ExactArgs(1),
			Long:    strings.TrimSpace(showLongHelp),
			Example: strings.TrimSpace(showExamples),
		},
	}

	cmd.Run = cmd.run

	return &cmd
}

func (c *showCmd) run(cmd *cobra.Command, args []string) {
	buildID, err := strconv.Atoi(args[0])
	if err == nil {
		c.showBuild(buildID)
	} else {
		c.showApp(args[0])
	}
}

func mustWriteStringSliceRows(fmt format.Formatter, header string, indentlvl int, sl []string) {
	defRowArgs := make([]interface{}, 0, indentlvl+1+1)

	for i := 0; i < indentlvl; i++ {
		defRowArgs = append(defRowArgs, "")
	}

	for i, val := range sl {
		var rowArgs []interface{}

		if i == 0 {
			rowArgs = append(defRowArgs, header)
		} else {
			rowArgs = append(defRowArgs, "")
		}

		if i+1 < len(sl) {
			val += ", "
		}
		rowArgs = append(rowArgs, term.Highlight(val))

		mustWriteRow(fmt, rowArgs...)
	}
}

func (c *showCmd) showApp(arg string) {
	repo := mustFindRepository()
	app := mustArgToApp(repo, arg)

	tasks := app.Tasks()
	baur.SortTasksByID(tasks)

	formatter := table.New(nil, stdout)

	mustWriteRow(formatter, "Application Name:", term.Highlight(app.Name), "", "")
	mustWriteRow(formatter, "Path:", term.Highlight(app.RelPath), "")

	mustWriteRow(formatter, "", "", "", "")
	for taskIdx, task := range tasks {
		mustWriteRow(formatter, term.Underline("Task"))
		mustWriteRow(formatter, "", "Name:", term.Highlight(task.Name), "", "")
		mustWriteRow(formatter, "", "Command:", term.Highlight(task.Command), "", "")

		if task.HasInputs() {
			mustWriteRow(formatter, "", "", "", "")
			mustWriteRow(formatter, "", term.Underline("Inputs:"), "", "")

			for i, f := range task.UnresolvedInputs.Files {
				mustWriteRow(formatter, "", "", "Type:", term.Highlight("File"))
				mustWriteStringSliceRows(formatter, "Paths:", 2, f.Paths)

				if i+1 < len(task.UnresolvedInputs.Files) {
					mustWriteRow(formatter, "", "", "", "")
				}
			}

			if len(task.UnresolvedInputs.Files) > 0 && len(task.UnresolvedInputs.GitFiles) > 0 {
				mustWriteRow(formatter, "", "", "", "")
			}

			for i, g := range task.UnresolvedInputs.GitFiles {
				mustWriteRow(formatter, "", "", "Type:", term.Highlight("GitFile"))
				mustWriteStringSliceRows(formatter, "Paths:", 2, g.Paths)

				if i+1 < len(task.UnresolvedInputs.GitFiles) {
					mustWriteRow(formatter, "", "", "", "")
				}
			}

			if len(task.UnresolvedInputs.GolangSources) > 0 &&
				len(task.UnresolvedInputs.GitFiles) > 0 || len(task.UnresolvedInputs.Files) > 0 {
				mustWriteRow(formatter, "", "", "", "")
			}

			for i, gs := range task.UnresolvedInputs.GolangSources {
				mustWriteRow(formatter, "", "", "", "")
				mustWriteRow(formatter, "", "", "Type:", term.Highlight("GolangSources"))
				mustWriteStringSliceRows(formatter, "Queries:", 2, gs.Queries)
				mustWriteStringSliceRows(formatter, "Environment:", 2, gs.Environment)
				mustWriteStringSliceRows(formatter, "BuildFlags:", 2, gs.BuildFlags)
				mustWriteRow(formatter, "", "", "Tests:", term.Highlight(gs.Tests))

				if i+1 < len(task.UnresolvedInputs.GolangSources) {
					mustWriteRow(formatter, "", "", "", "")
				}
			}
		}

		if task.HasOutputs() {
			mustWriteRow(formatter, "", term.Underline("Outputs:"), "", "")
		}

		for i, di := range task.Outputs.DockerImage {
			mustWriteRow(formatter, "", "", "Type:", term.Highlight("Docker Image"))
			mustWriteRow(formatter, "", "", "IDFile:", term.Highlight(di.IDFile))
			mustWriteRow(formatter, "", "", "Registry:", term.Highlight(di.RegistryUpload.Registry))
			mustWriteRow(formatter, "", "", "Repository:", term.Highlight(di.RegistryUpload.Repository))
			mustWriteRow(formatter, "", "", "Tag:", term.Highlight(di.RegistryUpload.Tag))

			if i+1 < len(task.Outputs.DockerImage) {
				mustWriteRow(formatter, "", "", "", "")
			}
		}

		for i, file := range task.Outputs.File {
			if len(task.Outputs.DockerImage) > 0 {
				mustWriteRow(formatter, "", "", "", "")
			}

			mustWriteRow(formatter, "", "", "Type:", term.Highlight("File"))
			mustWriteRow(formatter, "", "", "Path:", term.Highlight(file.Path))

			if !file.FileCopy.IsEmpty() {
				mustWriteRow(formatter, "", "", "Filecopy Destination:", term.Highlight(file.FileCopy.Path))
			}

			if !file.S3Upload.IsEmpty() {
				mustWriteRow(formatter, "", "", "S3 Bucket:", term.Highlight(file.S3Upload.Bucket))
				mustWriteRow(formatter, "", "", "S3 Key:", term.Highlight(file.S3Upload.Key))
			}

			if i+1 < len(task.Outputs.File) {
				mustWriteRow(formatter, "", "", "", "")
			}
		}

		if taskIdx+1 < len(tasks) {
			mustWriteRow(formatter, "", "", "", "")
		}
	}

	err := formatter.Flush()
	exitOnErr(err)
}

func vcsStr(v *storage.TaskRun) string {
	if len(v.VCSRevision) == 0 {
		return ""
	}

	if v.VCSIsDirty {
		return fmt.Sprintf("%s-dirty", v.VCSRevision)
	}

	return v.VCSRevision
}

func (c *showCmd) showBuild(taskRunID int) {
	repo := mustFindRepository()
	storageClt := mustNewCompatibleStorage(repo)

	taskRun, err := storageClt.TaskRun(ctx, taskRunID)
	if err != nil {
		if err == storage.ErrNotExist {
			log.Fatalf("task run with id %d does not exist\n", taskRunID)
		}

		exitOnErr(err)
	}

	outputs, err := storageClt.Outputs(ctx, taskRun.ID)
	if err != nil && !errors.Is(err, storage.ErrNotExist) {
		exitOnErr(err)
	}

	formatter := table.New(nil, stdout)

	mustWriteRow(formatter, "Run-ID:", term.Highlight(taskRun.ID))
	mustWriteRow(formatter, "Application:", term.Highlight(taskRun.ApplicationName))
	mustWriteRow(formatter, "Task:", term.Highlight(taskRun.TaskName))

	if taskRun.Result == storage.ResultSuccess {
		mustWriteRow(formatter, "Result", term.GreenHighlight(taskRun.Result))
	} else {
		mustWriteRow(formatter, "Result", term.RedHighlight(taskRun.Result))
	}

	mustWriteRow(formatter, "Started At:", term.Highlight(taskRun.StartTimestamp))
	mustWriteRow(
		formatter,
		"Build Duration:",
		term.Highlight(
			term.FormatDuration(
				taskRun.StopTimestamp.Sub(taskRun.StartTimestamp),
			),
		),
	)

	mustWriteRow(formatter, "Git Commit:", term.Highlight(vcsStr(&taskRun.TaskRun)))

	mustWriteRow(formatter, "Total Input Digest:", term.Highlight(taskRun.TotalInputDigest))
	mustWriteRow(formatter, "Output Count:", term.Highlight(len(outputs)))

	if len(outputs) > 0 {
		mustWriteRow(formatter)
		mustWriteRow(formatter, term.Underline("Outputs:"))
	}

	for i, o := range outputs {
		mustWriteRow(formatter, "", "Local Path:", term.Highlight(o.Name))
		mustWriteRow(formatter, "", "Digest:", term.Highlight(o.Digest))
		mustWriteRow(
			formatter,
			"",
			"Size:",
			term.Highlight(term.FormatSize(o.SizeBytes)),
		)
		mustWriteRow(formatter, "", "Type:", term.Highlight(o.Type))

		mustWriteRow(formatter)
		mustWriteRow(formatter, "", term.Underline("Uploads:"))

		for uploadIdx, upload := range o.Uploads {
			mustWriteRow(formatter, "", "", "URI:", term.Highlight(upload.URI))
			mustWriteRow(
				formatter,
				"",
				"",
				"Upload Duration:",
				term.FormatDuration(
					upload.UploadStopTimestamp.Sub(
						upload.UploadStartTimestamp),
				),
			)
			mustWriteRow(
				formatter,
				"", "", "Upload Method:", term.Highlight(upload.Method),
			)

			if uploadIdx+1 < len(o.Uploads) {
				mustWriteRow(formatter)
			}
		}

		if i+1 < len(outputs) {
			mustWriteRow(formatter)
		}
	}

	err = formatter.Flush()
	exitOnErr(err)
}
