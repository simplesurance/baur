package command

// TODO: adapt naming to task run change

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/spf13/cobra"

	"github.com/simplesurance/baur/format"
	"github.com/simplesurance/baur/format/table"
	"github.com/simplesurance/baur/internal/command/terminal"
	"github.com/simplesurance/baur/log"
	"github.com/simplesurance/baur/storage"
)

const showLongHelp = `
Show information about an application or a build.

If the name or the path to an application directory is passed,
application information are shown.
If a numeric build ID is passed, information about the build are shown.
`

const showExamples = `
baur show calc		show information about the calc application
baur show ui/shop	show information about the app in the ui/shop directory
baur show 512		show information about build 512
`

var showCmd = &cobra.Command{
	Use:     "show APP|APP-PATH|BUILD-ID",
	Short:   "show information about apps or builds",
	Args:    cobra.ExactArgs(1),
	Run:     show,
	Long:    strings.TrimSpace(showLongHelp),
	Example: strings.TrimSpace(showExamples),
}

func init() {
	rootCmd.AddCommand(showCmd)
}

func show(cmd *cobra.Command, args []string) {
	buildID, err := strconv.Atoi(args[0])
	if err == nil {
		showBuild(buildID)
	} else {
		showApp(args[0])
	}
}

func showApp(arg string) {
	var formatter format.Formatter

	// TODO: show all tasks of the app

	repo := MustFindRepository()
	app := mustArgToApp(repo, arg)

	formatter = table.New(nil, os.Stdout)

	mustWriteRow(formatter, []interface{}{terminal.Underline("General:")})
	mustWriteRow(formatter, []interface{}{"", "Application Name:", terminal.Highlight(app.Name)})
	mustWriteRow(formatter, []interface{}{"", "Path:", terminal.Highlight(app.RelPath)})

	stderr.Println("Showing task information not implemented")
	os.Exit(1)

	//mustWriteRow(formatter, []interface{}{"", "Build Command:", terminal.Highlight(task.Command)})

	// TODO: make this work again, print the outputs:
	/*
			outputs, err := task.Outputs()
			exitOnErr(err)

			if len(outputs) != 0 {
				mustWriteRow(formatter, []interface{}{})
				mustWriteRow(formatter, []interface{}{terminal.Underline("Outputs:")})

				for i, art := range outputs {
					mustWriteRow(formatter, []interface{}{"", "Type:", terminal.Highlight(art.Type())})
					mustWriteRow(formatter, []interface{}{"", "Local:", terminal.Highlight(art.String())})
					mustWriteRow(formatter, []interface{}{"", "Remote:", terminal.Highlight(art.UploadDestination())})

					if i+1 < len(outputs) {
						mustWriteRow(formatter, []interface{}{})
					}
				}
			}

		if task.HasInputs() {
			mustWriteRow(formatter, []interface{}{})
			mustWriteRow(formatter, []interface{}{terminal.Underline("Inputs:")})

			if len(task.UnresolvedInputs.Files.Paths) > 0 {
				mustWriteRow(formatter, []interface{}{})

				mustWriteRow(formatter, []interface{}{"", "Type:", terminal.Highlight("File")})
				mustWriteRow(formatter, []interface{}{"",
					"Paths:", terminal.Highlight(strings.Join(task.UnresolvedInputs.Files.Paths, ", ")),
				})

			}

			if len(task.UnresolvedInputs.GitFiles.Paths) > 0 {
				mustWriteRow(formatter, []interface{}{})

				mustWriteRow(formatter, []interface{}{"", "Type:", terminal.Highlight("GitFile")})
				mustWriteRow(formatter, []interface{}{"",
					"Paths:", terminal.Highlight(strings.Join(task.UnresolvedInputs.GitFiles.Paths, ", "))})
			}

			if len(task.UnresolvedInputs.GolangSources.Paths) > 0 {
				mustWriteRow(formatter, []interface{}{})

				mustWriteRow(formatter, []interface{}{"", "Type:", terminal.Highlight("GolangSources")})
				mustWriteRow(formatter, []interface{}{"",
					"Paths:", terminal.Highlight(strings.Join(task.UnresolvedInputs.GolangSources.Paths, ", "))})
				mustWriteRow(formatter, []interface{}{"",
					"Environment:", terminal.Highlight(strings.Join(task.UnresolvedInputs.GolangSources.Environment, ", "))})
			}
		}
	*/

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

func showBuild(taskRunID int) {
	var formatter format.Formatter

	repo := MustFindRepository()
	storageClt := mustNewCompatibleStorage(repo)

	taskRun, err := storageClt.TaskRun(ctx, taskRunID)
	if err != nil {
		if err == storage.ErrNotExist {
			log.Fatalf("task run with id %d does not exist\n", taskRunID)
		}

		log.Fatalln(err)
	}

	outputs, err := storageClt.Outputs(ctx, taskRun.ID)
	exitOnErr(err)

	formatter = table.New(nil, os.Stdout)

	mustWriteRow(formatter, []interface{}{terminal.Underline("General:")})
	mustWriteRow(formatter, []interface{}{"", "Application:", terminal.Highlight(taskRun.ApplicationName)})
	mustWriteRow(formatter, []interface{}{"", "Task:", terminal.Highlight(taskRun.TaskName)})
	mustWriteRow(formatter, []interface{}{"", "Run-ID:", terminal.Highlight(taskRun.ID)})

	mustWriteRow(formatter, []interface{}{"", "Started At:", terminal.Highlight(taskRun.StartTimestamp)})
	mustWriteRow(formatter, []interface{}{
		"",
		"Build Duration:",
		terminal.Highlight(fmt.Sprintf("%.2f s", taskRun.StopTimestamp.Sub(taskRun.StartTimestamp).Seconds())),
	})

	mustWriteRow(formatter, []interface{}{"", "Git Commit:", terminal.Highlight(vcsStr(&taskRun.TaskRun))})

	mustWriteRow(formatter, []interface{}{"", "Total Input Digest:", terminal.Highlight(taskRun.TotalInputDigest)})

	if len(outputs) > 0 {
		mustWriteRow(formatter, []interface{}{})
		mustWriteRow(formatter, []interface{}{terminal.Underline("Outputs:")})
	}

	for i, o := range outputs {
		mustWriteRow(formatter, []interface{}{"", "Local Path:", terminal.Highlight(o.Name)})
		mustWriteRow(formatter, []interface{}{"", "Digest:", terminal.Highlight(o.Digest)})
		mustWriteRow(formatter, []interface{}{
			"",
			"Size:",
			terminal.Highlight(bytesToMib(o.SizeBytes) + " MiB"),
		})
		mustWriteRow(formatter, []interface{}{"", "Type:", terminal.Highlight(o.Type)})

		mustWriteRow(formatter, []interface{}{})
		mustWriteRow(formatter, []interface{}{"", terminal.Underline("Uploads:")})

		for uploadIdx, upload := range o.Uploads {
			mustWriteRow(formatter, []interface{}{"", "", "URI:", terminal.Highlight(upload.URI)})
			mustWriteRow(formatter, []interface{}{
				"",
				"",
				"Upload Duration:",
				terminal.Highlight(
					durationToStrSeconds(upload.UploadStopTimestamp.Sub(upload.UploadStartTimestamp)) + " s"),
			})
			mustWriteRow(formatter, []interface{}{"", "", "Upload Method:", terminal.Highlight(upload.Method)})

			if uploadIdx+1 < len(o.Uploads) {
				mustWriteRow(formatter, []interface{}{})
			}
		}

		if i+1 < len(outputs) {
			mustWriteRow(formatter, []interface{}{})
		}
	}

	err = formatter.Flush()
	exitOnErr(err)
}
