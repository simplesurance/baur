package command

import (
	"fmt"
	"os"
	"path"

	"github.com/fatih/color"

	"github.com/simplesurance/baur"
	"github.com/simplesurance/baur/format"
	"github.com/simplesurance/baur/log"
	"github.com/simplesurance/baur/storage"
	"github.com/simplesurance/baur/storage/postgres"
)

// highlight is a function that highlights parts of strings in the cli output
var (
	greenHighlight  = color.New(color.FgGreen).SprintFunc()
	redHighlight    = color.New(color.FgRed).SprintFunc()
	yellowHighlight = color.New(color.FgYellow).SprintFunc()
	highlight       = greenHighlight
)

// MustFindRepository must find repo
func MustFindRepository() *baur.Repository {
	log.Debugln("searching for repository root...")

	rep, err := baur.FindRepository()
	if err != nil {
		if os.IsNotExist(err) {
			log.Fatalf("could not find repository root config file "+
				"ensure the file '%s' exist in the root\n",
				baur.RepositoryCfgFile)
		}

		log.Fatalln(err)
	}

	log.Debugf("repository root found: %v\n", rep.Path)

	return rep
}

func isAppDir(arg string) bool {
	cfgPath := path.Join(arg, baur.AppCfgFile)
	_, err := os.Stat(cfgPath)
	if err == nil {
		return true
	}

	return false
}

func mustArgToApp(repo *baur.Repository, arg string) *baur.App {
	if isAppDir(arg) {
		app, err := repo.AppByDir(arg)
		if err != nil {
			log.Fatalf("could not find application in dir '%s': %s\n", arg, err)
		}

		return app
	}

	app, err := repo.AppByName(arg)
	if err != nil {
		if os.IsNotExist(err) {
			log.Fatalf("could not find application with name '%s'\n", arg)
		}
		log.Fatalln(err)
	}

	return app
}

// MustGetPostgresClt must return the PG client
func MustGetPostgresClt(r *baur.Repository) *postgres.Client {
	clt, err := postgres.New(r.PSQLURL)
	if err != nil {
		log.Fatalf("could not establish connection to postgreSQL db: %s\n", err)
	}

	return clt
}

func mustGetCommitID(r *baur.Repository) string {
	commitID, err := r.GitCommitID()
	if err != nil {
		log.Fatalln(err)
	}

	return commitID
}

func mustGetGitWorktreeIsDirty(r *baur.Repository) bool {
	isDirty, err := r.GitWorkTreeIsDirty()
	if err != nil {
		log.Fatalln(err)
	}

	return isDirty
}

func vcsStr(v *storage.VCSState) string {
	if len(v.CommitID) == 0 {
		return ""
	}

	if v.IsDirty {
		return fmt.Sprintf("%s-dirty", v.CommitID)
	}

	return v.CommitID
}

func mustArgToApps(repo *baur.Repository, args []string) []*baur.App {
	if len(args) == 0 {
		apps, err := repo.FindApps()
		if err != nil {
			log.Fatalln(err)
		}

		if len(apps) == 0 {
			log.Fatalf("could not find any applications\n"+
				"- ensure the [Discover] section is correct in %s\n"+
				"- ensure that you have >1 application dirs "+
				"containing a %s file\n",
				repo.CfgPath, baur.AppCfgFile)
		}

		return apps
	}

	apps := make([]*baur.App, 0, len(args))
	for _, arg := range args {
		apps = append(apps, mustArgToApp(repo, arg))
	}

	return apps
}

func mustWriteRow(fmt format.Formatter, row []interface{}) {
	err := fmt.WriteRow(row)
	if err != nil {
		log.Fatalln(err)
	}
}

func fmtVertTitle(csvFormat bool, title string) string {
	if csvFormat {
		return title
	}

	return highlight(title + ":")
}
