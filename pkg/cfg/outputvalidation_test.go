package cfg

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOutputValidate(t *testing.T) {
	tests := []struct {
		name    string
		out     Output
		wantErr string
	}{
		{"empty output is valid", Output{}, ""},
		{
			"file with s3 and copy",
			Output{File: []FileOutput{{
				Path:     "a.bin",
				S3Upload: []S3Upload{{Bucket: "b", Key: "k"}},
				FileCopy: []FileCopy{{Path: "/tmp"}},
			}}},
			"",
		},
		{
			"docker image with registry upload",
			Output{DockerImage: []DockerImageOutput{{
				IDFile:         "id",
				RegistryUpload: []DockerImageRegistryUpload{{Repository: "r", Tag: "t"}},
			}}},
			"",
		},
		{"file path empty", Output{File: []FileOutput{{}}}, "File.path: can not be empty"},
		{
			"s3 key empty",
			Output{File: []FileOutput{{Path: "a", S3Upload: []S3Upload{{Bucket: "b"}}}}},
			"S3Upload.key: can not be empty",
		},
		{
			"s3 bucket empty",
			Output{File: []FileOutput{{Path: "a", S3Upload: []S3Upload{{Key: "k"}}}}},
			"S3Upload.bucket: can not be empty",
		},
		{
			"file copy path empty",
			Output{File: []FileOutput{{Path: "a", FileCopy: []FileCopy{{}}}}},
			"Filecopy.path: can not be empty",
		},
		{"docker idfile empty", Output{DockerImage: []DockerImageOutput{{}}}, "DockerImage.idfile: can not be empty"},
		{
			"docker repository empty",
			Output{DockerImage: []DockerImageOutput{{IDFile: "i", RegistryUpload: []DockerImageRegistryUpload{{Tag: "t"}}}}},
			"RegistryUpload.repository: can not be empty",
		},
		{
			"docker tag empty",
			Output{DockerImage: []DockerImageOutput{{IDFile: "i", RegistryUpload: []DockerImageRegistryUpload{{Repository: "r"}}}}},
			"RegistryUpload.tag: can not be empty",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := outputValidate(&tc.out)

			if tc.wantErr == "" {
				require.NoError(t, err)
				return
			}

			require.ErrorContains(t, err, tc.wantErr)
		})
	}
}

func TestOutputResolveResolvesAllFields(t *testing.T) {
	out := Output{
		File: []FileOutput{{
			Path:     "p",
			FileCopy: []FileCopy{{Path: "c"}},
			S3Upload: []S3Upload{{Bucket: "b", Key: "k"}},
		}},
		DockerImage: []DockerImageOutput{{
			IDFile:         "i",
			RegistryUpload: []DockerImageRegistryUpload{{Registry: "reg", Repository: "r", Tag: "t"}},
		}},
	}

	require.NoError(t, out.Resolve(upperResolver))

	assert.Equal(t, "P", out.File[0].Path)
	assert.Equal(t, "C", out.File[0].FileCopy[0].Path)
	assert.Equal(t, S3Upload{Bucket: "B", Key: "K"}, out.File[0].S3Upload[0])
	assert.Equal(t, "I", out.DockerImage[0].IDFile)
	assert.Equal(t, DockerImageRegistryUpload{Registry: "REG", Repository: "R", Tag: "T"}, out.DockerImage[0].RegistryUpload[0])
}

func TestOutputResolveErrorsContainFieldPath(t *testing.T) {
	r := failingResolverOn("bad")

	tests := []struct {
		name    string
		out     Output
		wantErr string
	}{
		{"file path", Output{File: []FileOutput{{Path: "bad"}}}, "FileOutput.path"},
		{"file copy", Output{File: []FileOutput{{Path: "ok", FileCopy: []FileCopy{{Path: "bad"}}}}}, "FileCopy.path"},
		{"s3 bucket", Output{File: []FileOutput{{Path: "ok", S3Upload: []S3Upload{{Bucket: "bad"}}}}}, "S3Upload.bucket"},
		{"s3 key", Output{File: []FileOutput{{Path: "ok", S3Upload: []S3Upload{{Key: "bad"}}}}}, "S3Upload.key"},
		{"docker idfile", Output{DockerImage: []DockerImageOutput{{IDFile: "bad"}}}, "DockerImage.idfile"},
		{
			"docker registry",
			Output{DockerImage: []DockerImageOutput{{IDFile: "ok", RegistryUpload: []DockerImageRegistryUpload{{Registry: "bad"}}}}},
			"RegistryUpload.registry",
		},
		{
			"docker repository",
			Output{DockerImage: []DockerImageOutput{{IDFile: "ok", RegistryUpload: []DockerImageRegistryUpload{{Repository: "bad"}}}}},
			"RegistryUpload.repository",
		},
		{
			"docker tag",
			Output{DockerImage: []DockerImageOutput{{IDFile: "ok", RegistryUpload: []DockerImageRegistryUpload{{Tag: "bad"}}}}},
			"RegistryUpload.tag",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.out.Resolve(r)

			require.ErrorContains(t, err, tc.wantErr)
		})
	}
}

func TestOutputMergeAppends(t *testing.T) {
	out := Output{File: []FileOutput{{Path: "a"}}}
	other := &OutputInclude{
		File:        []FileOutput{{Path: "b"}},
		DockerImage: []DockerImageOutput{{IDFile: "i"}},
	}

	out.Merge(other)

	assert.Equal(t, []FileOutput{{Path: "a"}, {Path: "b"}}, out.File)
	assert.Equal(t, []DockerImageOutput{{IDFile: "i"}}, out.DockerImage)
}

func TestOutputIncludeValidate(t *testing.T) {
	valid := func() *OutputInclude {
		return &OutputInclude{IncludeID: "id", File: []FileOutput{{Path: "a"}}}
	}

	require.NoError(t, valid().validate())

	o := valid()
	o.IncludeID = ""
	require.ErrorContains(t, o.validate(), "is empty")

	o = valid()
	o.File = nil
	require.ErrorContains(t, o.validate(), "no output is defined")

	o = valid()
	o.File[0].Path = ""
	require.ErrorContains(t, o.validate(), "path: can not be empty")
}

func TestOutputIncludeCloneIsIndependent(t *testing.T) {
	orig := &OutputInclude{
		IncludeID: "id",
		File:      []FileOutput{{Path: "a", S3Upload: []S3Upload{{Bucket: "b", Key: "k"}}}},
		filepath:  "/inc.toml",
	}

	clone := orig.clone()
	clone.File[0].Path = "changed"
	clone.File[0].S3Upload[0].Key = "changed"

	assert.Equal(t, "a", orig.File[0].Path)
	assert.Equal(t, "k", orig.File[0].S3Upload[0].Key)
	assert.Equal(t, "/inc.toml", clone.filepath, "the unexported filepath must survive the copy")
}

func TestInputIncludeValidate(t *testing.T) {
	require.NoError(t, (&InputInclude{IncludeID: "id"}).validate(), "an include without inputs is valid")
	require.NoError(t, (&InputInclude{IncludeID: "id", Files: []FileInputs{{Paths: []string{"a"}}}}).validate())

	require.ErrorContains(t, (&InputInclude{}).validate(), "is empty")
	require.ErrorContains(t, (&InputInclude{IncludeID: "a#b"}).validate(), "contains invalid character '#'")
	require.ErrorContains(t, (&InputInclude{IncludeID: "  "}).validate(), "only whitespaces")

	err := (&InputInclude{IncludeID: "id", Files: []FileInputs{{Paths: []string{""}}}}).validate()
	require.ErrorContains(t, err, "can not be empty")
}

func TestInputIncludeCloneIsIndependent(t *testing.T) {
	orig := &InputInclude{
		IncludeID: "id",
		Files:     []FileInputs{{Paths: []string{"a"}}},
		filepath:  "/inc.toml",
	}

	clone := orig.clone()
	clone.Files[0].Paths[0] = "changed"

	assert.Equal(t, "a", orig.Files[0].Paths[0])
	assert.Equal(t, "/inc.toml", clone.filepath)
}
