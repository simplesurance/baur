package cfg

import (
	"errors"
	"fmt"
	"io/ioutil"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/pelletier/go-toml"
)

const (
	includeIDSep   = "#"
	includeSpecFmt = "<RELATIVE-FILEPATH>" + includeIDSep + "<INCLUDE-ID>"
)

var (
	includeSpecRegex    = regexp.MustCompile(`(?m)[^` + includeIDSep + `]+[^` + includeIDSep + `#]+`)
	whitespaceOnlyRegex = regexp.MustCompile(`^\s+$`)
)

type Include struct {
	Input  InputIncludes
	Output OutputIncludes
	Task   TaskIncludes

	filePath string
}

// IncludeToFile marshals the Include struct to TOML and writes it to filepath.
func (incl *Include) IncludeToFile(filepath string) error {
	return toFile(incl, filepath, false)
}

// IncludeFromFile unmarshals an Include struct from a file.
func IncludeFromFile(path string) (*Include, error) {
	config := Include{}

	content, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}

	err = toml.Unmarshal(content, &config)
	if err != nil {
		return nil, err
	}

	config.filePath = path

	config.Output.RemoveEmptyElements()

	for _, task := range config.Task {
		OutputRemoveEmptySections(&task.Output)
	}

	return &config, err
}

// ExampleInclude returns an Include struct with exemplary values.
func ExampleInclude(id string) *Include {
	return &Include{
		Input: []*InputInclude{
			{
				IncludeID: id + "_input",
				Files: FileInputs{
					Paths: []string{"dbmigrations/*.sql"},
				},
				GitFiles: GitFileInputs{
					Paths: []string{"Makefile"},
				},
				GolangSources: GolangSources{
					Paths:       []string{"."},
					Environment: []string{"GOFLAGS=-mod=vendor", "GO111MODULE=on"},
				},
			},
		},
		Output: []*OutputInclude{
			{
				IncludeID: id + "_output",
				File: []*FileOutput{
					{
						Path: "dist/$APPNAME.tar.xz",
						S3Upload: S3Upload{
							Bucket:   "go-artifacts/",
							DestFile: "$APPNAME-$GITCOMMIT.tar.xz",
						},
						FileCopy: FileCopy{
							Path: "/mnt/fileserver/build_artifacts/$APPNAME-$GITCOMMIT.tar.xz",
						},
					},
				},
				DockerImage: []*DockerImageOutput{
					{
						IDFile: fmt.Sprintf("$APPNAME-container.id"),
						RegistryUpload: DockerImageRegistryUpload{
							Repository: "my-company/$APPNAME",
							Tag:        "$GITCOMMIT",
						},
					},
				},
			},
		},
		Task: TaskIncludes{
			{
				IncludeID: id + "_task_cbuild",
				Name:      "build",
				Command:   "make",
				Input: Input{
					GitFiles: GitFileInputs{
						Paths: []string{"*.c", "*.h", "Makefile"},
					},
				},
				Output: Output{
					File: []*FileOutput{
						{
							Path: "a.out",
							FileCopy: FileCopy{
								Path: "/artifacts",
							},
						},
					},
				},
			},
		},
	}
}

func validateIncludes(includes []string) error {
	// IncludeDB ensures during loading that include IDs are unique across
	// and in the same file. It is not validated by this function.
	for _, in := range includes {
		if filepath.IsAbs(in) {
			return NewFieldError("include specifier is an absolute path, must be a repository relative path", in)
		}

		if !includeSpecRegex.MatchString(in) {
			return NewFieldError("invalid include specifier, must be in format "+includeSpecFmt, in)
		}
	}

	return nil
}

func validateIncludeID(id string) error {
	if id == "" {
		return errors.New("is empty")
	}

	if strings.Contains(id, includeIDSep) {
		return errors.New("contains invalid character '#'")
	}

	if whitespaceOnlyRegex.MatchString(id) {
		return errors.New("contains only whitespaces, must contain non-whitespace characters")
	}

	return nil
}
