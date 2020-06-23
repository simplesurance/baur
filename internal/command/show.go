package command

import (
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/spf13/cobra"

	"github.com/simplesurance/baur"
	"github.com/simplesurance/baur/format/table"
	"github.com/simplesurance/baur/internal/command/term"
	"github.com/simplesurance/baur/log"
	"github.com/simplesurance/baur/storage"
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

func (c *showCmd) showApp(arg string) {
	repo := MustFindRepository()
	app := mustArgToApp(repo, arg)

	tasks := app.Tasks()
	baur.SortTasksByID(tasks)

	formatter := table.New(nil, stdout)

	mustWriteRowVa(formatter, "Application Name:", term.Highlight(app.Name), "", "")
	mustWriteRowVa(formatter, "Path:", term.Highlight(app.RelPath), "")

	mustWriteRowVa(formatter, "", "", "", "")
	for taskIdx, task := range tasks {
		mustWriteRowVa(formatter, term.Underline("Task"))
		mustWriteRowVa(formatter, "", "Name:", term.Highlight(task.Name), "", "")
		mustWriteRowVa(formatter, "", "Command:", term.Highlight(task.Command), "", "")

		if task.HasInputs() {
			mustWriteRowVa(formatter, "", "", "", "")
			mustWriteRowVa(formatter, "", term.Underline("Inputs:"), "", "")

			if len(task.UnresolvedInputs.Files.Paths) > 0 {
				mustWriteRowVa(formatter, "", "", "Type:", term.Highlight("File"))
				mustWriteRowVa(
					formatter,
					"",
					"",
					"Paths:",
					term.Highlight(strings.Join(task.UnresolvedInputs.Files.Paths, ", ")),
				)
			}

			if len(task.UnresolvedInputs.GitFiles.Paths) > 0 {
				if len(task.UnresolvedInputs.Files.Paths) > 0 {
					mustWriteRowVa(formatter, "", "", "", "")
				}

				mustWriteRowVa(formatter, "", "", "Type:", term.Highlight("GitFile"))
				mustWriteRowVa(
					formatter,
					"",
					"",
					"Paths:",
					term.Highlight(strings.Join(task.UnresolvedInputs.GitFiles.Paths, ", ")),
				)
			}

			if len(task.UnresolvedInputs.GolangSources.Paths) > 0 {
				if len(task.UnresolvedInputs.GitFiles.Paths) > 0 {
					mustWriteRowVa(formatter, "", "", "", "")
				}

				mustWriteRowVa(formatter, "", "", "Type:", term.Highlight("GolangSources"))
				mustWriteRowVa(
					formatter,
					"",
					"",
					"Paths:",
					term.Highlight(strings.Join(task.UnresolvedInputs.GolangSources.Paths, ", ")),
				)
				mustWriteRowVa(
					formatter,
					"",
					"",
					"Environment:", term.Highlight(strings.Join(task.UnresolvedInputs.GolangSources.Environment, ", ")),
				)
			}
		}

		if task.HasOutputs() {
			mustWriteRowVa(formatter, "", term.Underline("Outputs:"), "", "")
		}

		for i, di := range task.Outputs.DockerImage {
			mustWriteRowVa(formatter, "", "", "Type:", term.Highlight("Docker Image"))
			mustWriteRowVa(formatter, "", "", "IDFile:", term.Highlight(di.IDFile))
			mustWriteRowVa(formatter, "", "", "Registry:", term.Highlight(di.RegistryUpload.Registry))
			mustWriteRowVa(formatter, "", "", "Repository:", term.Highlight(di.RegistryUpload.Repository))
			mustWriteRowVa(formatter, "", "", "Tag:", term.Highlight(di.RegistryUpload.Tag))

			if i+1 < len(task.Outputs.DockerImage) {
				mustWriteRowVa(formatter, "", "", "", "")
			}
		}

		for i, file := range task.Outputs.File {
			if len(task.Outputs.DockerImage) > 0 {
				mustWriteRowVa(formatter, "", "", "", "")
			}

			mustWriteRowVa(formatter, "", "", "Type:", term.Highlight("File"))
			mustWriteRowVa(formatter, "", "", "Path:", term.Highlight(file.Path))

			if !file.FileCopy.IsEmpty() {
				mustWriteRowVa(formatter, "", "", "Filecopy Destination:", term.Highlight(file.FileCopy.Path))
			}

			if !file.S3Upload.IsEmpty() {
				mustWriteRowVa(formatter, "", "", "S3 Bucket:", term.Highlight(file.S3Upload.Bucket))
				mustWriteRowVa(formatter, "", "", "S3 Destfile:", term.Highlight(file.S3Upload.DestFile))
			}

			if i+1 < len(task.Outputs.File) {
				mustWriteRowVa(formatter, "", "", "", "")
			}
		}

		if taskIdx+1 < len(tasks) {
			mustWriteRowVa(formatter, "", "", "", "")
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
	repo := MustFindRepository()
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

	mustWriteRowVa(formatter, "Run-ID:", term.Highlight(taskRun.ID))
	mustWriteRowVa(formatter, "Application:", term.Highlight(taskRun.ApplicationName))
	mustWriteRowVa(formatter, "Task:", term.Highlight(taskRun.TaskName))

	if taskRun.Result == storage.ResultSuccess {
		mustWriteRowVa(formatter, "Result", term.GreenHighlight(taskRun.Result))
	} else {
		mustWriteRowVa(formatter, "Result", term.RedHighlight(taskRun.Result))
	}

	mustWriteRowVa(formatter, "Started At:", term.Highlight(taskRun.StartTimestamp))
	mustWriteRowVa(
		formatter,
		"Build Duration:",
		term.Highlight(
			term.FormatDuration(
				taskRun.StopTimestamp.Sub(taskRun.StartTimestamp),
			),
		),
	)

	mustWriteRowVa(formatter, "Git Commit:", term.Highlight(vcsStr(&taskRun.TaskRun)))

	mustWriteRowVa(formatter, "Total Input Digest:", term.Highlight(taskRun.TotalInputDigest))
	mustWriteRowVa(formatter, "Output Count:", term.Highlight(len(outputs)))

	if len(outputs) > 0 {
		mustWriteRowVa(formatter)
		mustWriteRowVa(formatter, term.Underline("Outputs:"))
	}

	for i, o := range outputs {
		mustWriteRowVa(formatter, "", "Local Path:", term.Highlight(o.Name))
		mustWriteRowVa(formatter, "", "Digest:", term.Highlight(o.Digest))
		mustWriteRowVa(
			formatter,
			"",
			"Size:",
			term.Highlight(term.FormatSize(o.SizeBytes)),
		)
		mustWriteRowVa(formatter, "", "Type:", term.Highlight(o.Type))

		mustWriteRowVa(formatter)
		mustWriteRowVa(formatter, "", term.Underline("Uploads:"))

		for uploadIdx, upload := range o.Uploads {
			mustWriteRowVa(formatter, "", "", "URI:", term.Highlight(upload.URI))
			mustWriteRowVa(
				formatter,
				"",
				"",
				"Upload Duration:",
				term.FormatDuration(
					upload.UploadStopTimestamp.Sub(
						upload.UploadStartTimestamp),
				),
			)
			mustWriteRowVa(
				formatter,
				"", "", "Upload Method:", term.Highlight(upload.Method),
			)

			if uploadIdx+1 < len(o.Uploads) {
				mustWriteRowVa(formatter)
			}
		}

		if i+1 < len(outputs) {
			mustWriteRowVa(formatter)
		}
	}

	err = formatter.Flush()
	exitOnErr(err)
}
