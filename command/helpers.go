package command

import (
	"os"
	"path"

	"github.com/simplesurance/baur"
	"github.com/simplesurance/baur/log"
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
