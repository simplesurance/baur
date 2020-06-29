package repotest

import (
	"io/ioutil"
	"os"
	"path"
	"testing"

	"github.com/simplesurance/baur"
	"github.com/simplesurance/baur/cfg"
	"github.com/simplesurance/baur/testutils"
	"github.com/simplesurance/baur/testutils/fstest"
)

func (r *Repo) CreateSimpleApp(t *testing.T) *cfg.App {
	t.Helper()

	appName := "simpleApp"

	app := cfg.App{
		Name: appName,
		Tasks: []*cfg.Task{
			{
				Name:    "build",
				Command: "build.sh",
				Input: cfg.Input{
					Files: cfg.FileInputs{
						Paths: []string{"build.sh", "output_content.txt"},
					},
				},
				Output: cfg.Output{
					File: []*cfg.FileOutput{
						{
							Path: "artifact",
							FileCopy: cfg.FileCopy{
								Path: r.FilecopyArtifactDir,
							},
						},
					},
				},
			},

			{
				Name:    "check",
				Command: "check.sh",
				Input: cfg.Input{
					GitFiles: cfg.GitFileInputs{
						Paths: []string{"check.sh"},
					},
				},
			},
		},
	}

	appDir := path.Join(r.Dir, appName)

	if err := os.Mkdir(appDir, 0775); err != nil {
		t.Fatal(err)
	}

	if err := app.ToFile(path.Join(appDir, baur.AppCfgFile)); err != nil {
		t.Fatalf("writing app cfg file failed: %s", err)
	}

	r.AppCfgs = append(r.AppCfgs, &app)

	buildFilePath := path.Join(path.Join(appDir, "build.sh"))
	checkFilePath := path.Join(path.Join(appDir, "check.sh"))

	fstest.WriteToFile(t, []byte(`
#!/bin/sh

echo "building app"
cat output_content.txt > output
`),
		buildFilePath)

	fstest.Chmod(t, buildFilePath, os.ModePerm)

	fstest.WriteToFile(t, []byte("1"), path.Join(appDir, "output_content.text"))

	fstest.WriteToFile(t, []byte(`
#!/bin/sh

echo "check successful"
`),
		checkFilePath)

	fstest.Chmod(t, checkFilePath, os.ModePerm)

	return &app
}

type Repo struct {
	AppCfgs             []*cfg.App
	Dir                 string
	FilecopyArtifactDir string
}

type repoOptions struct {
	keepTmpDir bool
}

type Opt func(*repoOptions)

func OptKeepTmpDir() Opt {
	return func(o *repoOptions) {
		o.keepTmpDir = true
	}
}

func CreateBaurRepository(t *testing.T, opts ...Opt) *Repo {
	t.Helper()

	var options repoOptions

	for _, opt := range opts {
		opt(&options)
	}

	tempDir, err := ioutil.TempDir("", "baur-filesrc-test")
	if err != nil {
		t.Fatal(err)
	}

	if !options.keepTmpDir {
		t.Cleanup(func() { os.RemoveAll(tempDir) })
	}

	artifactDir := path.Join(tempDir, "filecopy-artifacts")

	t.Logf("creating baur repository in %s", tempDir)

	if err := os.Mkdir(artifactDir, os.ModePerm); err != nil {
		t.Fatal(err)
	}

	cfgR := cfg.Repository{
		ConfigVersion: cfg.Version,

		Discover: cfg.Discover{
			Dirs:        []string{"."},
			SearchDepth: 10,
		},

		Database: cfg.Database{
			PGSQLURL: testutils.PSQLURL(),
		},
	}

	if err := cfgR.ToFile(path.Join(tempDir, baur.RepositoryCfgFile), false); err != nil {
		t.Fatalf("could not write repository cfg file: %s", err)
	}

	// TODO: do `git init` and add files to repo

	return &Repo{
		Dir:                 tempDir,
		FilecopyArtifactDir: artifactDir,
	}
}
