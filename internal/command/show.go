package command

import (
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/spf13/cobra"

	"github.com/simplesurance/baur"
	"github.com/simplesurance/baur/format/table"
	"github.com/simplesurance/baur/internal/command/terminal"
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

	mustWriteRowVa(formatter, "Application Name:", terminal.Highlight(app.Name), "", "")
	mustWriteRowVa(formatter, "Path:", terminal.Highlight(app.RelPath), "")

	mustWriteRowVa(formatter, "", "", "", "")
	for taskIdx, task := range tasks {
		mustWriteRowVa(formatter, terminal.Underline("Task"))
		mustWriteRowVa(formatter, "", "Name:", terminal.Highlight(task.Name), "", "")
		mustWriteRowVa(formatter, "", "Command:", terminal.Highlight(task.Command), "", "")

		if task.HasInputs() {
			mustWriteRowVa(formatter, "", "", "", "")
			mustWriteRowVa(formatter, "", terminal.Underline("Inputs:"), "", "")

			if len(task.UnresolvedInputs.Files.Paths) > 0 {
				mustWriteRowVa(formatter, "", "", "Type:", terminal.Highlight("File"))
				mustWriteRowVa(
					formatter,
					"",
					"",
					"Paths:",
					terminal.Highlight(strings.Join(task.UnresolvedInputs.Files.Paths, ", ")),
				)
			}

			if len(task.UnresolvedInputs.GitFiles.Paths) > 0 {
				if len(task.UnresolvedInputs.Files.Paths) > 0 {
					mustWriteRowVa(formatter, "", "", "", "")
				}

				mustWriteRowVa(formatter, "", "", "Type:", terminal.Highlight("GitFile"))
				mustWriteRowVa(
					formatter,
					"",
					"",
					"Paths:",
					terminal.Highlight(strings.Join(task.UnresolvedInputs.GitFiles.Paths, ", ")),
				)
			}

			if len(task.UnresolvedInputs.GolangSources.Paths) > 0 {
				if len(task.UnresolvedInputs.GitFiles.Paths) > 0 {
					mustWriteRowVa(formatter, "", "", "", "")
				}

				mustWriteRowVa(formatter, "", "", "Type:", terminal.Highlight("GolangSources"))
				mustWriteRowVa(
					formatter,
					"",
					"",
					"Paths:",
					terminal.Highlight(strings.Join(task.UnresolvedInputs.GolangSources.Paths, ", ")),
				)
				mustWriteRowVa(
					formatter,
					"",
					"",
					"Environment:", terminal.Highlight(strings.Join(task.UnresolvedInputs.GolangSources.Environment, ", ")),
				)
			}
		}

		if task.HasOutputs() {
			mustWriteRowVa(formatter, "", terminal.Underline("Outputs:"), "", "")
		}

		for i, di := range task.Outputs.DockerImage {
			mustWriteRowVa(formatter, "", "", "Type:", terminal.Highlight("Docker Image"))
			mustWriteRowVa(formatter, "", "", "IDFile:", terminal.Highlight(di.IDFile))
			mustWriteRowVa(formatter, "", "", "Registry:", terminal.Highlight(di.RegistryUpload.Registry))
			mustWriteRowVa(formatter, "", "", "Repository:", terminal.Highlight(di.RegistryUpload.Repository))
			mustWriteRowVa(formatter, "", "", "Tag:", terminal.Highlight(di.RegistryUpload.Tag))

			if i+1 < len(task.Outputs.DockerImage) {
				mustWriteRowVa(formatter, "", "", "", "")
			}
		}

		for i, file := range task.Outputs.File {
			if len(task.Outputs.DockerImage) > 0 {
				mustWriteRowVa(formatter, "", "", "", "")
			}

			mustWriteRowVa(formatter, "", "", "Type:", terminal.Highlight("File"))
			mustWriteRowVa(formatter, "", "", "Path:", terminal.Highlight(file.Path))

			if !file.FileCopy.IsEmpty() {
				mustWriteRowVa(formatter, "", "", "Filecopy Destination:", terminal.Highlight(file.FileCopy.Path))
			}

			if !file.S3Upload.IsEmpty() {
				mustWriteRowVa(formatter, "", "", "S3 Bucket:", terminal.Highlight(file.S3Upload.Bucket))
				mustWriteRowVa(formatter, "", "", "S3 Destfile:", terminal.Highlight(file.S3Upload.DestFile))
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

	mustWriteRowVa(formatter, "Run-ID:", terminal.Highlight(taskRun.ID))
	mustWriteRowVa(formatter, "Application:", terminal.Highlight(taskRun.ApplicationName))
	mustWriteRowVa(formatter, "Task:", terminal.Highlight(taskRun.TaskName))

	if taskRun.Result == storage.ResultSuccess {
		mustWriteRowVa(formatter, "Result", terminal.GreenHighlight(taskRun.Result))
	} else {
		mustWriteRowVa(formatter, "Result", terminal.RedHighlight(taskRun.Result))
	}

	mustWriteRowVa(formatter, "Started At:", terminal.Highlight(taskRun.StartTimestamp))
	mustWriteRowVa(
		formatter,
		"Build Duration:",
		terminal.Highlight(fmt.Sprintf("%.2f s", taskRun.StopTimestamp.Sub(taskRun.StartTimestamp).Seconds())),
	)

	mustWriteRowVa(formatter, "Git Commit:", terminal.Highlight(vcsStr(&taskRun.TaskRun)))

	mustWriteRowVa(formatter, "Total Input Digest:", terminal.Highlight(taskRun.TotalInputDigest))
	mustWriteRowVa(formatter, "Output Count:", terminal.Highlight(len(outputs)))

	if len(outputs) > 0 {
		mustWriteRowVa(formatter)
		mustWriteRowVa(formatter, terminal.Underline("Outputs:"))
	}

	for i, o := range outputs {
		mustWriteRowVa(formatter, "", "Local Path:", terminal.Highlight(o.Name))
		mustWriteRowVa(formatter, "", "Digest:", terminal.Highlight(o.Digest))
		mustWriteRowVa(
			formatter,
			"",
			"Size:",
			terminal.Highlight(terminal.BytesToMib(o.SizeBytes)+" MiB"),
		)
		mustWriteRowVa(formatter, "", "Type:", terminal.Highlight(o.Type))

		mustWriteRowVa(formatter)
		mustWriteRowVa(formatter, "", terminal.Underline("Uploads:"))

		for uploadIdx, upload := range o.Uploads {
			mustWriteRowVa(formatter, "", "", "URI:", terminal.Highlight(upload.URI))
			mustWriteRowVa(
				formatter,
				"",
				"",
				"Upload Duration:",
				terminal.Highlight(
					terminal.DurationToStrSeconds(upload.UploadStopTimestamp.Sub(upload.UploadStartTimestamp))+" s"),
			)
			mustWriteRowVa(
				formatter,
				"", "", "Upload Method:", terminal.Highlight(upload.Method),
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
