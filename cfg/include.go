package cfg

import (
	"io/ioutil"

	"github.com/pelletier/go-toml"
	"github.com/pkg/errors"
)

// Include represents an include configuration file.
type Include struct {
	BuildInput  []BuildInput
	BuildOutput []BuildOutput
}

// ExampleInclude returns an Include struct with exemplary values.
func ExampleInclude() *Include {
	return &Include{
		BuildInput: []BuildInput{
			{
				GolangSources: GolangSources{
					Paths:       []string{"."},
					Environment: []string{"GOFLAGS=-mod=vendor", "GO111MODULE=on"},
				},
				Files: FileInputs{
					Paths: []string{".app.toml"},
				},
			},
			{
				Files: FileInputs{
					Paths: []string{".app.toml"},
				},
				GitFiles: GitFileInputs{
					Paths: []string{"Makefile", "*.c", "include/*.h"},
				},
			},
		},
		BuildOutput: []BuildOutput{
			{
				File: []*FileOutput{
					{
						Path: "dist/dist.tar.xz",
						S3Upload: S3Upload{
							Bucket:   "go-artifacts/",
							DestFile: "$APPNAME-$GITCOMMIT.tar.xz",
						},
						FileCopy: FileCopy{
							Path: "/mnt/fileserver/build_artifacts/$APPNAME-$GITCOMMIT.tar.xz",
						},
					},
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

// Validate validates an Include configuration struct.
func (in *Include) Validate() error {
	for _, bi := range in.BuildInput {
		if err := bi.Validate(); err != nil {
			return errors.Wrap(err, "[[BuildInput]] section contains errors")
		}
	}

	for _, bo := range in.BuildOutput {
		if err := bo.Validate(); err != nil {
			return errors.Wrap(err, "[[BuildOutput]] section contains errors")
		}
	}

	return nil
}
