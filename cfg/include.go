package cfg

import (
	"io/ioutil"

	"github.com/pelletier/go-toml"
)

// Include represents an include configuration file.
type Include struct {
	BuildInput []BuildInputInclude
}

// TODO: how to prevent duplicating the comment and name  tags here, how to reuse it from the BuildInput struct?

// BuildInputInclude contains information about an includeable BuildInput section.
type BuildInputInclude struct {
	ID            string        `toml:"id" comment:"Identifier to reference the include"`
	Files         FileInputs    `comment:"Inputs specified by file glob paths"`
	GitFiles      GitFileInputs `comment:"Inputs specified by path, matching only Git tracked files"`
	GolangSources GolangSources `comment:"Inputs specified by directories containing Golang applications"`
}

// ExampleInclude returns an Include struct with exemplary values.
func ExampleInclude() *Include {
	return &Include{
		BuildInput: []BuildInputInclude{
			{
				ID: "go_app_build_inputs",
				GolangSources: GolangSources{
					Paths:       []string{"."},
					Environment: []string{"GOFLAGS=-mod=vendor", "GO111MODULE=on"},
				},
				Files: FileInputs{
					Paths: []string{".app.toml"},
				},
			},
			{
				ID: "c_app_build_inputs",
				Files: FileInputs{
					Paths: []string{".app.toml"},
				},
				GitFiles: GitFileInputs{
					Paths: []string{"Makefile", "*.c", "include/*.h"},
				},
			},
		},
	}
}

// IncludeToFile serializes the Include struct to TOML and writes it to filepath.
func (in *Include) IncludeToFile(filepath string) error {
	return toFile(in, filepath, false)
}

// IncludeFromFile deserializes an Include struct from a file.
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

	return &config, err
}

// ToBuildInput returns a BuildInput struct with values set to the same then in the BuildInputInclude
func (bi *BuildInputInclude) ToBuildInput() BuildInput {
	return BuildInput{
		Files:         bi.Files,
		GitFiles:      bi.GitFiles,
		GolangSources: bi.GolangSources,
	}
}
