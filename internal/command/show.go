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
	"github.com/simplesurance/baur/v1/internal/fs"
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
baur show calc.build	show information about the build task of the calc application
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
			Use:     "show <APP-NAME>|<APP-PATH>|<APP-NAME.TASK-NAME>|<TASK-RUN-ID>",
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
	arg := args[0]

	buildID, err := strconv.Atoi(arg)
	if err == nil {
		c.showBuild(buildID)
		return
	}

	if isDir, _ := fs.IsDir(arg); isDir {
		c.showApp(arg)
		return
	}

	if strings.Contains(arg, ".") {
		c.showTask(arg)
		return
	}

	c.showApp(arg)
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

func (*showCmd) strCmd(cmd []string) string {
	var result strings.Builder

	for i, e := range cmd {
		result.WriteString(fmt.Sprintf("'%s'", e))
		if i < len(cmd)+1 {
			result.WriteRune(' ')
		}
	}

	return result.String()
}

func (c *showCmd) showApp(appName string) {
	formatter := table.New(nil, stdout)

	repo := mustFindRepository()
	app := mustArgToApp(repo, appName)

	tasks := app.Tasks()
	baur.SortTasksByID(tasks)

	mustWriteRow(formatter, "Application Name:", term.Highlight(app.Name), "", "")
	mustWriteRow(formatter, "Directory:", term.Highlight(app.RelPath), "")

	mustWriteRow(formatter, "", "", "", "")
	for taskIdx, task := range tasks {
		c.printTask(formatter, task)

		if taskIdx+1 < len(tasks) {
			mustWriteRow(formatter, "", "", "", "")
		}
	}

	err := formatter.Flush()
	exitOnErr(err)
}

func (c *showCmd) showTask(taskName string) {
	formatter := table.New(nil, stdout)

	repo := mustFindRepository()

	task := mustArgToTask(repo, taskName)

	c.printTask(formatter, task)

	err := formatter.Flush()
	exitOnErr(err)
}

func (c *showCmd) printTask(formatter format.Formatter, task *baur.Task) {
	mustWriteRow(formatter, term.Underline("Task"))
	mustWriteRow(formatter, "", "Name:", term.Highlight(task.Name), "", "")
	mustWriteRow(formatter, "", "Command:", term.Highlight(
		c.strCmd(task.Command),
	), "", "")
	mustWriteStringSliceRows(formatter, "Config Files:", 1, task.CfgFilepaths)

	if task.HasInputs() {
		mustWriteRow(formatter, "", "", "", "")
		mustWriteRow(formatter, "", term.Underline("Inputs:"), "", "")

		for i, f := range task.UnresolvedInputs.Files {
			mustWriteRow(formatter, "", "", "Type:", term.Highlight("File"))
			mustWriteRow(formatter, "", "", "Optional:", term.Highlight(f.Optional))
			mustWriteRow(formatter, "", "", "Git tracked only:", term.Highlight(f.GitTrackedOnly))
			mustWriteStringSliceRows(formatter, "Paths:", 2, f.Paths)

			if i+1 < len(task.UnresolvedInputs.Files) {
				mustWriteRow(formatter, "", "", "", "")
			}
		}

		if len(task.UnresolvedInputs.Files) > 0 && len(task.UnresolvedInputs.GolangSources) > 0 {
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
		mustWriteRow(formatter, "", "", "")
		mustWriteRow(formatter, "", "", term.Underline("Uploads:"), "", "")

		for i, dest := range di.RegistryUpload {
			mustWriteRow(formatter, "", "", "", "Registry:", term.Highlight(dest.Registry))
			mustWriteRow(formatter, "", "", "", "Repository:", term.Highlight(dest.Repository))
			mustWriteRow(formatter, "", "", "", "Tag:", term.Highlight(dest.Tag))

			if i+1 < len(di.RegistryUpload) {
				mustWriteRow(formatter, "", "", "", "", "")
			}
		}

		if i+1 < len(task.Outputs.DockerImage) {
			mustWriteRow(formatter, "", "", "", "", "")
		}
	}

	for i, file := range task.Outputs.File {
		if len(task.Outputs.DockerImage) > 0 {
			mustWriteRow(formatter, "", "", "", "")
		}

		mustWriteRow(formatter, "", "", "Type:", term.Highlight("File"))
		mustWriteRow(formatter, "", "", "Path:", term.Highlight(file.Path))
		mustWriteRow(formatter, "", "", "")

		mustWriteRow(formatter, "", "", term.Underline("Uploads:"), "", "")

		if len(file.FileCopy) > 0 {
			for i, fc := range file.FileCopy {
				mustWriteRow(formatter, "", "", "", "Filecopy Destination:", term.Highlight(fc.Path))

				if i+1 < len(file.FileCopy) {
					mustWriteRow(formatter, "", "", "", "", "")
				}
			}
		}

		if len(file.S3Upload) > 0 {
			if len(file.FileCopy) > 0 {
				mustWriteRow(formatter, "", "", "", "", "")
			}

			for i, s3 := range file.S3Upload {
				mustWriteRow(formatter, "", "", "", "S3 Bucket:", term.Highlight(s3.Bucket))
				mustWriteRow(formatter, "", "", "", "S3 Key:", term.Highlight(s3.Key))

				if i+1 < len(file.S3Upload) {
					mustWriteRow(formatter, "", "", "", "", "")
				}
			}
		}

		if i+1 < len(task.Outputs.File) {
			mustWriteRow(formatter, "", "", "", "")
		}
	}
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

func (*showCmd) showBuild(taskRunID int) {
	repo := mustFindRepository()
	storageClt := mustNewCompatibleStorage(repo)
	defer storageClt.Close()

	taskRun, err := storageClt.TaskRun(ctx, taskRunID)
	if err != nil {
		if err == storage.ErrNotExist {
			stderr.Printf("task run with id %d does not exist\n", taskRunID)
			exitFunc(1)
		}

		stderr.Println(err)
		exitFunc(1)
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
