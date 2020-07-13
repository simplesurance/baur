package gosource

import (
	"path"
	"testing"

	"github.com/simplesurance/baur/fs"
	"github.com/simplesurance/baur/internal/testutils/fstest"
	"github.com/simplesurance/baur/internal/testutils/strtest"
)

const testfileMainGo = `
package main

import (
	"fmt"

	"github.com/simplesurance/baur-test/generator"
)

func main() {
	fmt.Println(generator.RandomNumber())
}

`
const testfileGeneratorGo = `
package generator

import (
	"math/rand"
)

// RandomNumber returns a random number
func RandomNumber() int {
	return rand.Int()
}
`

const testFileGoMod = `
module github.com/simplesurance/baur-test
`

func createGoProject(t *testing.T, dir string, createGoModFile bool) (string, string, []string) {
	t.Helper()

	tmpdir := fstest.CreateTempDir(t)
	projectPath := path.Join(tmpdir, dir)
	generatorPkgPath := path.Join(projectPath, "generator")

	err := fs.Mkdir(generatorPkgPath)
	if err != nil {
		t.Fatal(err)
	}

	mainGoPath := path.Join(projectPath, "main.go")
	randomGenGoPath := path.Join(projectPath, "generator", "generator.go")

	fstest.WriteToFile(t, []byte(testfileMainGo), mainGoPath)
	fstest.WriteToFile(t, []byte(testfileGeneratorGo), randomGenGoPath)

	if createGoModFile {
		fstest.WriteToFile(t, []byte(testFileGoMod), path.Join(projectPath, "go.mod"))
	}

	return tmpdir, projectPath, []string{mainGoPath, randomGenGoPath}

}

func TestResolveWithGoPath(t *testing.T) {
	tmpdir, projectPath, filepaths := createGoProject(t, "src/github.com/simplesurance/baur-test/", false)

	resolver := NewResolver(
		nil,
	)

	resolvedFiles, err := resolver.Resolve([]string{"GOPATH=" + tmpdir}, projectPath)
	if err != nil {
		t.Fatal(err)
	}

	for _, path := range resolvedFiles {
		if !strtest.InSlice(filepaths, path) {
			t.Errorf("resolved files contain '%s' but it's not part of the application", path)
		}
	}

	for _, path := range filepaths {
		if !strtest.InSlice(resolvedFiles, path) {
			t.Errorf("resolved go source files are missing '%s'", path)
		}
	}

}

func TestResolveWithGoMod(t *testing.T) {
	_, projectPath, filepaths := createGoProject(t, "baur-test/", true)

	resolver := NewResolver(nil)
	resolvedFiles, err := resolver.Resolve(nil, projectPath)
	if err != nil {
		t.Fatal(err)
	}

	for _, path := range resolvedFiles {
		if !strtest.InSlice(filepaths, path) {
			t.Errorf("resolved files contain '%s' but it's not part of the application", path)
		}
	}

	for _, path := range filepaths {
		if !strtest.InSlice(resolvedFiles, path) {
			t.Errorf("resolved go source files are missing '%s'", path)
		}
	}

}
