package cfg

import (
	"errors"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type funcResolver func(string) (string, error)

func (f funcResolver) Resolve(s string) (string, error) { return f(s) }

var upperResolver = funcResolver(func(s string) (string, error) { return strings.ToUpper(s), nil })

func failingResolverOn(bad string) Resolver {
	return funcResolver(func(s string) (string, error) {
		if s == bad {
			return "", errors.New("resolve failed")
		}

		return s, nil
	})
}

func TestGolangSourcesValidate(t *testing.T) {
	tests := []struct {
		name    string
		gs      GolangSources
		wantErr string
	}{
		{"queries only", GolangSources{Queries: []string{"./..."}}, ""},
		{"all fields", GolangSources{Queries: []string{"."}, Environment: []string{"A=b"}, BuildFlags: []string{"-tags=x"}, Tests: true}, ""},
		{"empty section is valid", GolangSources{}, ""},
		{"environment without queries", GolangSources{Environment: []string{"A=b"}}, "must be set"},
		{"build flags without queries", GolangSources{BuildFlags: []string{"-x"}}, "must be set"},
		{"tests without queries", GolangSources{Tests: true}, "must be set"},
		{"empty query", GolangSources{Queries: []string{"ok", ""}}, "empty string is an invalid query"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.gs.validate()

			if tc.wantErr == "" {
				require.NoError(t, err)
				return
			}

			require.ErrorContains(t, err, tc.wantErr)
		})
	}
}

func TestGolangSourcesResolve(t *testing.T) {
	gs := GolangSources{
		Queries:     []string{"q1", "q2"},
		Environment: []string{"e"},
		BuildFlags:  []string{"f"},
	}

	require.NoError(t, gs.resolve(upperResolver))

	assert.Equal(t, []string{"Q1", "Q2"}, gs.Queries)
	assert.Equal(t, []string{"E"}, gs.Environment)
	assert.Equal(t, []string{"F"}, gs.BuildFlags)
}

func TestGolangSourcesResolveErrors(t *testing.T) {
	tests := []struct {
		name    string
		gs      GolangSources
		wantErr string
	}{
		{"environment", GolangSources{Environment: []string{"bad"}}, "Environment.bad"},
		{"query", GolangSources{Queries: []string{"bad"}}, "Paths.bad"},
		{"build flag", GolangSources{BuildFlags: []string{"bad"}}, "build_flags.bad"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.gs.resolve(failingResolverOn("bad"))

			require.ErrorContains(t, err, tc.wantErr)
		})
	}
}

func TestEnvVarsInputsValidate(t *testing.T) {
	require.NoError(t, (&EnvVarsInputs{}).Validate())
	require.NoError(t, (&EnvVarsInputs{Names: []string{"HOME", "BAUR_*", "a b"}}).Validate())

	err := (&EnvVarsInputs{Names: []string{"OK", ""}}).Validate()
	require.ErrorContains(t, err, "element can not be empty")

	err = (&EnvVarsInputs{Names: []string{"A=B"}}).Validate()
	require.ErrorContains(t, err, `"A=B" contains invalid character '='`)
}

func TestFileInputsAndExcludeListValidate(t *testing.T) {
	require.NoError(t, (&FileInputs{Paths: []string{"a", "**/*.go"}}).validate())
	require.ErrorContains(t, (&FileInputs{Paths: []string{"a", ""}}).validate(), "can not be empty")

	require.NoError(t, (&FileExcludeList{Paths: []string{"a"}}).Validate())
	require.ErrorContains(t, (&FileExcludeList{Paths: []string{""}}).Validate(), "can not be empty")
}

func TestFileInputsAndExcludeListResolve(t *testing.T) {
	f := FileInputs{Paths: []string{"a", "b"}}
	require.NoError(t, f.resolve(upperResolver))
	assert.Equal(t, []string{"A", "B"}, f.Paths)
	require.ErrorContains(t, f.resolve(failingResolverOn("A")), "Paths.A")

	e := FileExcludeList{Paths: []string{"x"}}
	require.NoError(t, e.resolve(upperResolver))
	assert.Equal(t, []string{"X"}, e.Paths)
	require.ErrorContains(t, e.resolve(failingResolverOn("X")), "Paths.X")
}

func TestTaskInfoValidate(t *testing.T) {
	require.NoError(t, (&TaskInfo{TaskName: "build", EnvVarName: "BUILD_INFO"}).Validate())

	err := (&TaskInfo{TaskName: "build", EnvVarName: ""}).Validate()
	require.ErrorContains(t, err, "env_var: can not be empty")

	for _, name := range []string{"", "a.b", "a,b", "a*", "a#b", " a"} {
		t.Run("task name "+name, func(t *testing.T) {
			err := (&TaskInfo{TaskName: name, EnvVarName: "X"}).Validate()
			require.ErrorContains(t, err, "task_name")
		})
	}
}

func TestValidateTaskInfosReportsTaskName(t *testing.T) {
	err := validateTaskInfos([]TaskInfo{
		{TaskName: "ok", EnvVarName: "A"},
		{TaskName: "broken", EnvVarName: ""},
	})

	require.ErrorContains(t, err, "TaskInfo[broken]")
}

func TestInputIsEmpty(t *testing.T) {
	assert.True(t, (&Input{}).IsEmpty())

	for name, in := range map[string]Input{
		"files":    {Files: []FileInputs{{}}},
		"golang":   {GolangSources: []GolangSources{{}}},
		"env":      {EnvironmentVariables: []EnvVarsInputs{{}}},
		"excluded": {ExcludedFiles: FileExcludeList{Paths: []string{"a"}}},
		"taskinfo": {TaskInfos: []TaskInfo{{}}},
	} {
		t.Run(name, func(t *testing.T) {
			assert.False(t, in.IsEmpty())
		})
	}

	assert.True(t, (&InputInclude{}).IsEmpty())
	assert.False(t, (&InputInclude{TaskInfos: []TaskInfo{{}}}).IsEmpty())
}

func TestInputMergeAppendsAllSections(t *testing.T) {
	in := Input{
		Files:                []FileInputs{{Paths: []string{"a"}}},
		EnvironmentVariables: []EnvVarsInputs{{Names: []string{"A"}}},
		ExcludedFiles:        FileExcludeList{Paths: []string{"x"}},
	}
	other := &InputInclude{
		Files:                []FileInputs{{Paths: []string{"b"}}},
		GolangSources:        []GolangSources{{Queries: []string{"q"}}},
		EnvironmentVariables: []EnvVarsInputs{{Names: []string{"B"}}},
		ExcludedFiles:        FileExcludeList{Paths: []string{"y"}},
		TaskInfos:            []TaskInfo{{TaskName: "t", EnvVarName: "E"}},
	}

	in.merge(other)

	assert.Equal(t, []FileInputs{{Paths: []string{"a"}}, {Paths: []string{"b"}}}, in.Files)
	assert.Equal(t, []GolangSources{{Queries: []string{"q"}}}, in.GolangSources)
	assert.Equal(t, []EnvVarsInputs{{Names: []string{"A"}}, {Names: []string{"B"}}}, in.EnvironmentVariables)
	assert.Equal(t, []string{"x", "y"}, in.ExcludedFiles.Paths)
	assert.Equal(t, []TaskInfo{{TaskName: "t", EnvVarName: "E"}}, in.TaskInfos)
}

func TestInputResolveResolvesFilesExcludesAndGolangSources(t *testing.T) {
	in := Input{
		Files:         []FileInputs{{Paths: []string{"a"}}},
		ExcludedFiles: FileExcludeList{Paths: []string{"x"}},
		GolangSources: []GolangSources{{Queries: []string{"q"}}},
	}

	require.NoError(t, in.resolve(upperResolver))

	assert.Equal(t, []string{"A"}, in.Files[0].Paths)
	assert.Equal(t, []string{"X"}, in.ExcludedFiles.Paths)
	assert.Equal(t, []string{"Q"}, in.GolangSources[0].Queries)
}

func TestInputResolveErrorsContainSectionName(t *testing.T) {
	r := failingResolverOn("bad")

	err := (&Input{Files: []FileInputs{{Paths: []string{"bad"}}}}).resolve(r)
	require.ErrorContains(t, err, "Files")

	err = (&Input{ExcludedFiles: FileExcludeList{Paths: []string{"bad"}}}).resolve(r)
	require.ErrorContains(t, err, "ExcludedFiles")

	err = (&Input{GolangSources: []GolangSources{{Queries: []string{"bad"}}}}).resolve(r)
	require.ErrorContains(t, err, "GoLangSources")
}

func TestInputValidateReportsSection(t *testing.T) {
	tests := []struct {
		name    string
		in      Input
		wantErr string
	}{
		{"valid", Input{Files: []FileInputs{{Paths: []string{"a"}}}}, ""},
		{"files", Input{Files: []FileInputs{{Paths: []string{""}}}}, "Files"},
		{"golang", Input{GolangSources: []GolangSources{{Tests: true}}}, "GolangSources"},
		{"env", Input{EnvironmentVariables: []EnvVarsInputs{{Names: []string{"A=B"}}}}, "EnvVariables"},
		{"excluded", Input{ExcludedFiles: FileExcludeList{Paths: []string{""}}}, "ExcludedFiles"},
		{"taskinfo", Input{TaskInfos: []TaskInfo{{TaskName: "", EnvVarName: "E"}}}, "TaskInfos"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := inputValidate(&tc.in)

			if tc.wantErr == "" {
				require.NoError(t, err)
				return
			}

			require.ErrorContains(t, err, tc.wantErr)
		})
	}
}
