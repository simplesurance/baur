package command

import (
	"fmt"
	"os"
	"path"
	"time"

	"github.com/simplesurance/baur"
	"github.com/simplesurance/baur/log"
	"github.com/simplesurance/baur/storage/postgres"
)

func mustFindRepository() *baur.Repository {
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

func mustFindApps(r *baur.Repository) []*baur.App {
	apps, err := r.FindApps()
	if err != nil {
		log.Fatalln(err)
	}

	if len(apps) == 0 {
		log.Fatalf("could not find any applications\n"+
			"- ensure the [Discover] section is correct in %s\n"+
			"- ensure that you have >1 application dirs "+
			"containing a %s file\n",
			r.CfgPath, baur.AppCfgFile)
	}

	return apps

}

func mustGetPostgresClt(r *baur.Repository) *postgres.Client {
	clt, err := postgres.New(r.PSQLURL)
	if err != nil {
		log.Fatalf("could not establish connection to postgreSQL db: %s", err)
	}

	return clt
}

func durationToStrSec(d time.Duration) string {
	return fmt.Sprintf("%.2fs", d.Seconds())
}
