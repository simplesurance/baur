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

	formatter = table.New(nil, os.Stdout)

	mustWriteRow(formatter, []interface{}{underline("General:")})
	mustWriteRow(formatter, []interface{}{"", "Name:", highlight(app.Name)})
	mustWriteRow(formatter, []interface{}{"", "Path:", highlight(app.RelPath)})
	mustWriteRow(formatter, []interface{}{"", "Build Command:", highlight(app.BuildCmd)})

	if len(app.Outputs) != 0 {
		mustWriteRow(formatter, []interface{}{})
		mustWriteRow(formatter, []interface{}{underline("Outputs:")})

		for i, art := range app.Outputs {
			mustWriteRow(formatter, []interface{}{"", "Type:", highlight(art.Type())})
			mustWriteRow(formatter, []interface{}{"", "Local:", highlight(art.String())})
			mustWriteRow(formatter, []interface{}{"", "Remote:", highlight(art.UploadDestination())})

			if i+1 < len(app.Outputs) {
				mustWriteRow(formatter, []interface{}{})
			}
		}
	}

	if app.HasBuildInputs() {
		var printNewLine bool

		mustWriteRow(formatter, []interface{}{})
		mustWriteRow(formatter, []interface{}{underline("Inputs:")})

		if len(app.UnresolvedInputs.Files.Paths) > 0 {
			mustWriteRow(formatter, []interface{}{"", "Type:", highlight("File")})
			mustWriteRow(formatter, []interface{}{"",
				"Paths:", highlight(strings.Join(app.UnresolvedInputs.Files.Paths, ", ")),
			})

			printNewLine = true

		}

		if len(app.UnresolvedInputs.GitFiles.Paths) > 0 {
			if printNewLine {
				mustWriteRow(formatter, []interface{}{})
			}

			mustWriteRow(formatter, []interface{}{"", "Type:", highlight("GitFile")})
			mustWriteRow(formatter, []interface{}{"",
				"Paths:", highlight(strings.Join(app.UnresolvedInputs.GitFiles.Paths, ", "))})

			printNewLine = true
		}

		if len(app.UnresolvedInputs.GolangSources.Paths) > 0 {
			if printNewLine {
				mustWriteRow(formatter, []interface{}{})
			}

			mustWriteRow(formatter, []interface{}{"", "Type:", highlight("GolangSources")})
			mustWriteRow(formatter, []interface{}{"",
				"Paths:", highlight(strings.Join(app.UnresolvedInputs.GolangSources.Paths, ", "))})
			mustWriteRow(formatter, []interface{}{"",
				"Environment:", highlight(strings.Join(app.UnresolvedInputs.GolangSources.Environment, ", "))})
		}
	}

	if err := formatter.Flush(); err != nil {
		log.Fatalln(err)
	}
}

func showBuild(buildID int) {
	var formatter format.Formatter

	repo := MustFindRepository()
	storageClt := MustGetPostgresClt(repo)

	build, err := storageClt.GetBuildWithoutInputsOutputs(int(buildID))
	if err != nil {
		if err == storage.ErrNotExist {
			log.Fatalf("build with id %d does not exist\n", buildID)
		}

		log.Fatalln(err)
	}

	build.Outputs, err = storageClt.GetBuildOutputs(build.ID)
	if err != nil {
		log.Fatalln(err)
	}

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

	if err := formatter.Flush(); err != nil {
		log.Fatalln(err)
	}

}
