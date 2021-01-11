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

// ToFile marshals the Include struct to TOML and writes it to filepath.
func (incl *Include) ToFile(filepath string, opts ...ToFileOpt) error {
	return toFile(incl, filepath, opts...)
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

	config.setFilepaths(path)

	return &config, err
}

func (incl *Include) setFilepaths(path string) {
	incl.filePath = path

	for _, in := range incl.Input {
		in.filepath = path
	}

	for _, out := range incl.Output {
		out.filepath = path
	}

	for _, task := range incl.Task {
		task.cfgFiles = map[string]struct{}{path: {}}
	}
}

// validateUniqIncludeIDs validates that IDs of all Input, Output and Task
// includes are unique.
func (incl *Include) validateUniqIncludeIDs() error {
	uniqIncludeIDs := map[string]struct{}{}

	for _, in := range incl.Input {
		if _, exist := uniqIncludeIDs[in.IncludeID]; exist {
			return newFieldError(
				fmt.Sprintf("contains multiple includes with the includeID %q, includeIDs must be unique in a file", in.IncludeID),
				"Input", "include_id",
			)
		}

		uniqIncludeIDs[in.IncludeID] = struct{}{}
	}

	for _, out := range incl.Output {
		if _, exist := uniqIncludeIDs[out.IncludeID]; exist {
			return newFieldError(
				fmt.Sprintf("contains multiple includes with the includeID %q, includeIDs must be unique in a file", out.IncludeID),
				"Input", "include_id",
			)
		}

		uniqIncludeIDs[out.IncludeID] = struct{}{}
	}

	for _, task := range incl.Task {
		if _, exist := uniqIncludeIDs[task.IncludeID]; exist {
			return newFieldError(
				fmt.Sprintf("contains multiple includes with the includeID %q, includeIDs must be unique in a file", task.IncludeID),
				"Input", "include_id",
			)
		}

		uniqIncludeIDs[task.IncludeID] = struct{}{}
	}

	return nil

}

// ExampleInclude returns an Include struct with exemplary values.
func ExampleInclude(id string) *Include {
	return &Include{
		Input: []*InputInclude{
			{
				IncludeID: id + "_input",
				Files: []FileInputs{
					{
						Paths: []string{"dbmigrations/*.sql"},
					},
				},
				GitFiles: []GitFileInputs{
					{
						Paths: []string{"Makefile"},
					},
				},
				GolangSources: []GolangSources{
					{
						Queries:     []string{"."},
						Environment: []string{"GOFLAGS=-mod=vendor", "GO111MODULE=on"},
					},
				},
			},
		},
		Output: []*OutputInclude{
			{
				IncludeID: id + "_output",
				File: []FileOutput{
					{
						Path: "dist/$APPNAME.tar.xz",
						S3Upload: []S3Upload{
							{
								Bucket: "go-artifacts/",
								Key:    "$APPNAME-$GITCOMMIT.tar.xz",
							},
						},
						FileCopy: []FileCopy{
							{
								Path: "/mnt/fileserver/build_artifacts/$APPNAME-$GITCOMMIT.tar.xz",
							},
						},
					},
				},
				DockerImage: []DockerImageOutput{
					{
						IDFile: "$APPNAME-container.id",
						RegistryUpload: []DockerImageRegistryUpload{
							{
								Repository: "my-company/$APPNAME",
								Tag:        "$GITCOMMIT",
							},
						},
					},
				},
			},
		},
		Task: TaskIncludes{
			{
				IncludeID: id + "_task_cbuild",
				Name:      "build",
				Command:   []string{"make"},
				Input: Input{
					GitFiles: []GitFileInputs{
						{
							Paths: []string{"*.c", "*.h", "Makefile"},
						},
					},
				},
				Output: Output{
					File: []FileOutput{
						{
							Path: "a.out",
							FileCopy: []FileCopy{
								{
									Path: "/artifacts",
								},
							},
						},
					},
				},
			},
		},
	}
}

// validateIncludes validates includeSpecs
func validateIncludes(includes []string) error {
	for _, in := range includes {
		if filepath.IsAbs(in) {
			return newFieldError("include specifier is an absolute path, must be a repository relative path", in)
		}

		if !includeSpecRegex.MatchString(in) {
			return newFieldError("invalid include specifier, must be in format "+includeSpecFmt, in)
		}
	}

	return nil
}

// validateIncludeID validates the format of an include ID.
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
