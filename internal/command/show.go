package command

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/spf13/cobra"

	"github.com/simplesurance/baur/format"
	"github.com/simplesurance/baur/format/table"
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

	repo := MustFindRepository()
	app := mustArgToApp(repo, arg)
	task := app.Task()

	formatter = table.New(nil, os.Stdout)

	mustWriteRow(formatter, []interface{}{underline("General:")})
	mustWriteRow(formatter, []interface{}{"", "Name:", highlight(app.Name)})
	mustWriteRow(formatter, []interface{}{"", "Path:", highlight(app.RelPath)})
	mustWriteRow(formatter, []interface{}{"", "Build Command:", highlight(task.Command)})

	outputs, err := task.BuildOutputs()
	exitOnErr(err)

	if len(outputs) != 0 {
		mustWriteRow(formatter, []interface{}{})
		mustWriteRow(formatter, []interface{}{underline("Outputs:")})

		for i, art := range outputs {
			mustWriteRow(formatter, []interface{}{"", "Type:", highlight(art.Type())})
			mustWriteRow(formatter, []interface{}{"", "Local:", highlight(art.String())})
			mustWriteRow(formatter, []interface{}{"", "Remote:", highlight(art.UploadDestination())})

			if i+1 < len(outputs) {
				mustWriteRow(formatter, []interface{}{})
			}
		}
	}

	if task.HasInputs() {
		mustWriteRow(formatter, []interface{}{})
		mustWriteRow(formatter, []interface{}{underline("Inputs:")})

		if len(task.UnresolvedInputs.Files.Paths) > 0 {
			mustWriteRow(formatter, []interface{}{})

			mustWriteRow(formatter, []interface{}{"", "Type:", highlight("File")})
			mustWriteRow(formatter, []interface{}{"",
				"Paths:", highlight(strings.Join(task.UnresolvedInputs.Files.Paths, ", ")),
			})

		}

		if len(task.UnresolvedInputs.GitFiles.Paths) > 0 {
			mustWriteRow(formatter, []interface{}{})

			mustWriteRow(formatter, []interface{}{"", "Type:", highlight("GitFile")})
			mustWriteRow(formatter, []interface{}{"",
				"Paths:", highlight(strings.Join(task.UnresolvedInputs.GitFiles.Paths, ", "))})
		}

		if len(task.UnresolvedInputs.GolangSources.Paths) > 0 {
			mustWriteRow(formatter, []interface{}{})

			mustWriteRow(formatter, []interface{}{"", "Type:", highlight("GolangSources")})
			mustWriteRow(formatter, []interface{}{"",
				"Paths:", highlight(strings.Join(task.UnresolvedInputs.GolangSources.Paths, ", "))})
			mustWriteRow(formatter, []interface{}{"",
				"Environment:", highlight(strings.Join(task.UnresolvedInputs.GolangSources.Environment, ", "))})
		}
	}

	err = formatter.Flush()
	exitOnErr(err)
}

func showBuild(buildID int) {
	var formatter format.Formatter

	repo := MustFindRepository()
	storageClt := MustGetPostgresClt(repo)

	build, err := storageClt.GetBuildWithoutInputsOutputs(buildID)
	if err != nil {
		if err == storage.ErrNotExist {
			log.Fatalf("build with id %d does not exist\n", buildID)
		}

		log.Fatalln(err)
	}

	build.Outputs, err = storageClt.GetBuildOutputs(build.ID)
	exitOnErr(err)

	formatter = table.New(nil, os.Stdout)

	mustWriteRow(formatter, []interface{}{underline("General:")})
	mustWriteRow(formatter, []interface{}{"", "Name:", highlight(build.Application.Name)})
	mustWriteRow(formatter, []interface{}{"", "ID:", highlight(build.ID)})

	mustWriteRow(formatter, []interface{}{"", "Started At:", highlight(build.StartTimeStamp)})
	mustWriteRow(formatter, []interface{}{
		"",
		"Build Duration:",
		highlight(fmt.Sprintf("%.2f s", build.StopTimeStamp.Sub(build.StartTimeStamp).Seconds())),
	})

	mustWriteRow(formatter, []interface{}{"", "Git Commit:", highlight(vcsStr(&build.VCSState))})

	mustWriteRow(formatter, []interface{}{"", "Total Input Digest:", highlight(build.TotalInputDigest)})

	if len(build.Outputs) > 0 {
		mustWriteRow(formatter, []interface{}{})
		mustWriteRow(formatter, []interface{}{underline("Outputs:")})
	}
	for i, o := range build.Outputs {
		mustWriteRow(formatter, []interface{}{"", "URI:", highlight(o.Upload.URI)})
		mustWriteRow(formatter, []interface{}{"", "Digest:", highlight(o.Digest)})
		mustWriteRow(formatter, []interface{}{
			"",
			"Size:",
			highlight(bytesToMib(int(o.SizeBytes)) + " MiB"),
		})
		mustWriteRow(formatter, []interface{}{
			"",
			"Upload Duration:",
			highlight(durationToStrSeconds(o.Upload.UploadDuration) + " s"),
		})
		mustWriteRow(formatter, []interface{}{"", "Type:", highlight(o.Type)})
		mustWriteRow(formatter, []interface{}{"", "Upload Method:", highlight(o.Upload.Method)})

		if i+1 < len(build.Outputs) {
			mustWriteRow(formatter, []interface{}{})
		}
	}

	err = formatter.Flush()
	exitOnErr(err)
}
