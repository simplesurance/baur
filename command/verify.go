package command

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/simplesurance/baur/log"
	"github.com/simplesurance/baur/storage"
	"github.com/simplesurance/baur/term"
)

const verifyLongHelp = `
Check for issues in past builds.

The command scans past builds for patterns that indicate issues in the
Build.Input or Build.Output configuration of an application.
It finds builds for the same application that have the same digest for it's
inputs but produced different outputs.

Exit Codes:
0 - no issues found
1 - internal error
2 - issues found
`

const verifyExitCodeIssuesFound int = 2

var verifyFromDate string
var verifyCmd = &cobra.Command{
	Use:   "verify",
	Short: "check for issues in past builds",
	Long:  strings.TrimSpace(verifyLongHelp),
	Run:   verify,
}

func init() {
	verifyStartdate := time.Now().AddDate(0, -1, 0)
	verifyStartdateStr := fmt.Sprintf("%04d.%02d.%02d",
		verifyStartdate.Year(), verifyStartdate.Month(), verifyStartdate.Day())

	verifyCmd.Flags().StringVarP(&verifyFromDate, "from", "s", verifyStartdateStr,
		"start date, format: YYYY.MM.DD")

	rootCmd.AddCommand(verifyCmd)
}

func containsOnlyDockerIssues(issues []*storage.VerifyIssue) bool {
	for _, i := range issues {
		if i.Output.Type != storage.DockerArtifact {
			return false
		}
	}

	return true
}

func verify(cmd *cobra.Command, args []string) {
	const dateLayout = "2006.01.02"
	startTs, err := time.Parse(dateLayout, verifyFromDate)
	if err != nil {
		log.Fatalf("parsing start date value failed: %s:", err)
	}
	repo := MustFindRepository()

	clt := MustGetPostgresClt(repo)
	defer clt.Close()

	storedApps, err := clt.GetApps()
	if err != nil {
		if err == storage.ErrNotExist {
			log.Fatalln("database doesn't contain any build informations, run some builds first")
		}

		log.Fatalln("retrieving applications from storage failed:", err)
	}

	fmt.Printf("Scanning for builds after %s with same inputs that produced different outputs...\n", startTs.Format(dateLayout))

	var issuesFound bool
	for _, app := range storedApps {
		issues, err := storage.VerifySameInputDigestSameOutputs(clt, app.Name, startTs)
		if err != nil && err != storage.ErrNotExist {
			log.Fatalln("verifiying if builds with same input digests have the same outputs failed:", err)
		}

		// Docker images are not reproducible, timestamps in the
		// filesystem of the image change with every build, we
		// can't verify them  currently :/
		if len(issues) == 0 || containsOnlyDockerIssues(issues) {
			fmt.Printf("%s: %s\n", app.Name, greenHighlight("OK"))

			continue
		}

		fmt.Printf("%s: %s\n", app.Name, redHighlight("Issues found"))
		for _, i := range issues {
			issuesFound = true
			fmt.Printf("- %s: build %d and %d have same inputs, but digest of output %s differs\n",
				i.Issue, i.Build.ID, i.ReferenceBuild.ID, i.Output.Name)
		}

	}

	if issuesFound {
		term.PrintSep()
		fmt.Println(redHighlight("Issues found"))
		fmt.Printf("\nPossible reasons:\n")
		fmt.Println("- builds are not reproducible, ensure a builds with the same inputs produce outputs with the same digest")
		fmt.Println("- specified inputs of the build are incomplete")

		os.Exit(verifyExitCodeIssuesFound)
	}

	term.PrintSep()
	fmt.Println(greenHighlight("No issues found"))
}
