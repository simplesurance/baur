package repotest

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/simplesurance/baur/v3/internal/digest"
	"github.com/simplesurance/baur/v3/internal/digest/gitobjectid"
	"github.com/simplesurance/baur/v3/internal/fs"
	"github.com/simplesurance/baur/v3/internal/testutils/dbtest"
	"github.com/simplesurance/baur/v3/internal/testutils/fstest"
	"github.com/simplesurance/baur/v3/internal/testutils/gittest"
	"github.com/simplesurance/baur/v3/internal/testutils/ostest"
	"github.com/simplesurance/baur/v3/pkg/baur"
	"github.com/simplesurance/baur/v3/pkg/cfg"
)

func (r *Repo) CreateAppWithoutTasks(t *testing.T) *cfg.App {
	t.Helper()

	appName := "appWithoutIO"

	app := cfg.App{
		Name: appName,
	}

	appDir := filepath.Join(r.Dir, appName)

	if err := os.Mkdir(appDir, 0775); err != nil {
		t.Fatal(err)
	}

	if err := app.ToFile(filepath.Join(appDir, baur.AppCfgFile)); err != nil {
		t.Fatalf("writing app cfg file failed: %s", err)
	}

	r.AppCfgs = append(r.AppCfgs, &app)

	return &app
}

func (r *Repo) CreateSimpleApp(t *testing.T) *cfg.App {
	t.Helper()

	appName := "simpleApp"

	buildFile := "build.sh"
	if runtime.GOOS == "windows" {
		buildFile = "build.bat"
	}
	checkFile := "check.sh"
	if runtime.GOOS == "windows" {
		checkFile = "check.bat"
	}
	buildCommand := []string{"sh", fmt.Sprintf("./%s", buildFile)}
	if runtime.GOOS == "windows" {
		buildCommand = []string{"cmd", "/C", buildFile}
	}
	checkCommand := []string{"sh", fmt.Sprintf("./%s", checkFile)}
	if runtime.GOOS == "windows" {
		checkCommand = []string{"cmd", "/C", checkFile}
	}

	app := cfg.App{
		Name: appName,
		Tasks: []*cfg.Task{
			{
				Name:    "build",
				Command: buildCommand,
				Input: cfg.Input{
					Files: []cfg.FileInputs{
						{
							Paths: []string{buildFile, "output_content.txt"},
						},
					},
				},
				Output: cfg.Output{
					File: []cfg.FileOutput{
						{
							Path: "output",
							FileCopy: []cfg.FileCopy{
								{
									Path: r.FilecopyArtifactDir,
								},
							},
						},
					},
				},
			},

			{
				Name:    "check",
				Command: checkCommand,
				Input: cfg.Input{
					Files: []cfg.FileInputs{
						{
							Paths: []string{checkFile},
						},
					},
				},
			},
		},
	}

	appDir := filepath.Join(r.Dir, appName)

	if err := os.Mkdir(appDir, 0775); err != nil {
		t.Fatal(err)
	}

	if err := app.ToFile(filepath.Join(appDir, baur.AppCfgFile)); err != nil {
		t.Fatalf("writing app cfg file failed: %s", err)
	}

	r.AppCfgs = append(r.AppCfgs, &app)

	buildFilePath := filepath.Join(appDir, buildFile)
	checkFilePath := filepath.Join(appDir, checkFile)

	fstest.WriteToFile(t, []byte(`
#!/bin/sh

echo "building app"
more output_content.txt > output
`),
		buildFilePath)

	fstest.Chmod(t, buildFilePath, os.ModePerm)

	fstest.WriteToFile(t, []byte("1"), filepath.Join(appDir, "output_content.txt"))

	fstest.WriteToFile(t, []byte(`
#!/bin/sh

echo "check successful"
`),
		checkFilePath)

	fstest.Chmod(t, checkFilePath, os.ModePerm)

	return &app
}

func (r *Repo) CreateAppWithNoOutputs(t *testing.T, appName string) *cfg.App {
	t.Helper()

	inputFileName := fmt.Sprintf("%s.txt", appName)

	newCommandSlice := func() []string {
		if runtime.GOOS == "windows" {
			return []string{"cmd", "/C"}
		}

		return []string{}
	}

	buildCommand := append(newCommandSlice(), "echo", "build", appName)
	testCommand := append(newCommandSlice(), "echo", "test", appName)

	app := cfg.App{
		Name: appName,
		Tasks: []*cfg.Task{
			{
				Name:    "build",
				Command: buildCommand,
				Input: cfg.Input{
					Files: []cfg.FileInputs{
						{
							Paths: []string{"**"},
						},
					},
				},
			},
			{
				Name:    "test",
				Command: testCommand,
				Input: cfg.Input{
					Files: []cfg.FileInputs{
						{
							Paths: []string{"**"},
						},
					},
				},
			},
		},
	}

	appDir := filepath.Join(r.Dir, appName)

	if err := os.Mkdir(appDir, 0775); err != nil {
		t.Fatal(err)
	}

	if err := app.ToFile(filepath.Join(appDir, baur.AppCfgFile)); err != nil {
		t.Fatalf("writing app cfg file failed: %s", err)
	}

	r.AppCfgs = append(r.AppCfgs, &app)

	inputFilePath := filepath.Join(appDir, inputFileName)
	fstest.WriteToFile(t, []byte(appName), inputFilePath)

	return &app
}

func (r *Repo) WriteAdditionalFileContents(t *testing.T, appName, fileName, contents string) *digest.Digest {
	t.Helper()

	absPath := filepath.Join(r.Dir, appName, fileName)
	gobj := gitobjectid.New(r.Dir, t.Logf)
	file := baur.NewInputFile(absPath, filepath.Join(appName, fileName), false, baur.WithHashFn(gobj.File))
	fstest.WriteToFile(t, []byte(contents), absPath)

	digest, err := file.CalcDigest()
	if err != nil {
		t.Fatal(err)
	}

	return digest
}

type Repo struct {
	Cfg                 *cfg.Repository
	AppCfgs             []*cfg.App
	Dir                 string
	FilecopyArtifactDir string
}

// TaskIDs returns the tasks ids (<AppName>.<TaskName>) of all tasks in the AppCfgs slice
func (r *Repo) TaskIDs() []string {
	var result []string

	for _, appCfg := range r.AppCfgs {
		for _, task := range appCfg.Tasks {
			result = append(result, fmt.Sprintf("%s.%s", appCfg.Name, task.Name))
		}
	}

	return result
}

type repoOptions struct {
	keepTmpDir  bool
	createNewDB bool
}

type Opt func(*repoOptions)

func WithKeepTmpDir() Opt {
	return func(o *repoOptions) {
		o.keepTmpDir = true
	}
}

// WithNewDB create a new database with an unique name and use it for the baur
// repository.
func WithNewDB() Opt {
	return func(o *repoOptions) {
		o.createNewDB = true
	}
}

// CreateBaurRepository creates a new baur repository in a temporary directory
// and a new postgres database with a unique name.
// The funcion changes the current working directory to the created repository directory.
func CreateBaurRepository(t *testing.T, opts ...Opt) *Repo {
	t.Helper()

	var dbURL string
	var options repoOptions

	for _, opt := range opts {
		opt(&options)
	}

	if options.createNewDB {
		var err error

		dbName := dbtest.UniqueDBName()

		t.Logf("creating database %s", dbName)
		if dbURL, err = dbtest.CreateDB(dbName); err != nil {
			t.Fatalf("creating db failed: %s", err)
		}
	} else {
		dbURL = dbtest.PSQLURL()
	}

	t.Logf("database url: %q", dbURL)

	tempDir, err := os.MkdirTemp("", "baur-filesrc-test")
	if err != nil {
		t.Fatal(err)
	}

	tempDir, err = fs.RealPath(tempDir)
	if err != nil {
		t.Fatalf("canonicalizing temp dir path %q failed: %s", tempDir, err)
	}

	if !options.keepTmpDir {
		t.Cleanup(func() { os.RemoveAll(tempDir) })
	}

	artifactDir := filepath.Join(tempDir, "filecopy-artifacts")

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
			PGSQLURL: dbURL,
		},
	}

	if err := cfgR.ToFile(filepath.Join(tempDir, baur.RepositoryCfgFile)); err != nil {
		t.Fatalf("could not write repository cfg file: %s", err)
	}

	ostest.Chdir(t, tempDir)

	gittest.CreateRepository(t, tempDir)
	gittest.CommitFilesToGit(t, tempDir)

	t.Logf("changed working directory to baur repository: %q", tempDir)

	return &Repo{
		Cfg:                 &cfgR,
		Dir:                 tempDir,
		FilecopyArtifactDir: artifactDir,
	}
}
