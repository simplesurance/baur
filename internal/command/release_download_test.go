//go:build s3test

package command

// Tests only run on Linux because the Windows CI agent is not setup to provide
// the S3 test container:

import (
	"path/filepath"
	"testing"

	"github.com/simplesurance/baur/v3/internal/testutils/fstest"
	"github.com/simplesurance/baur/v3/internal/testutils/repotest"
	"github.com/simplesurance/baur/v3/internal/testutils/s3test"
	"github.com/simplesurance/baur/v3/pkg/baur"
	"github.com/simplesurance/baur/v3/pkg/cfg"

	"github.com/stretchr/testify/require"
)

func TestReleaseDownload(t *testing.T) {
	initTest(t)
	r := repotest.CreateBaurRepository(t, repotest.WithNewDB())
	runInitDb(t)

	appName := "app1"
	appCfg := cfg.App{
		Name: appName,
		Tasks: cfg.Tasks{
			{
				Name:    "build",
				Command: []string{"./build.sh"},
				Input: cfg.Input{
					Files: []cfg.FileInputs{
						{Paths: []string{"build.sh"}},
					},
				},
				Output: cfg.Output{
					File: []cfg.FileOutput{
						{
							Path: "output",
							S3Upload: []cfg.S3Upload{
								{
									Bucket: "mock",
									Key:    "build-output-file",
								},
								{
									Bucket: "mock",
									Key:    "build-output-file-cp",
								},
							},
						},
					},
				},
			},
			{
				Name:    "build1",
				Command: []string{"./build1.sh"},
				Input: cfg.Input{
					Files: []cfg.FileInputs{
						{Paths: []string{"build1.sh"}},
					},
				},
				Output: cfg.Output{
					File: []cfg.FileOutput{
						{
							Path: "output1",
							S3Upload: []cfg.S3Upload{
								{
									Bucket: "mock",
									Key:    "build-output-file",
								},
								{
									Bucket: "mock",
									Key:    "build-output-file-cp",
								},
							},
						},
					},
				},
			},
			{
				Name:    "no-outputs",
				Command: []string{"true"},
				Input: cfg.Input{
					Files: []cfg.FileInputs{
						{Paths: []string{"build.sh"}},
					},
				},
			},
		},
	}
	appDir := filepath.Join(r.Dir, appName)

	fstest.WriteExecutable(t, []byte("#!/bin/sh\necho o1 >output"), filepath.Join(appDir, "build.sh"))
	fstest.WriteExecutable(t, []byte("#!/bin/sh\necho o2 >output1"), filepath.Join(appDir, "build1.sh"))

	err := appCfg.ToFile(filepath.Join(appDir, baur.AppCfgFile))
	require.NoError(t, err)

	s3test.SetupEnv(t)

	runCmd := newRunCmd()
	require.NotPanics(t, func() { require.NoError(t, runCmd.Execute()) })

	releaseCmd := newReleaseCreateCmd()
	releaseCmd.SetArgs([]string{"all"})
	require.NotPanics(t, func() { require.NoError(t, releaseCmd.Execute()) })

	t.Run("downloadAll", func(t *testing.T) {
		initTest(t)

		downloadCmd := newReleaseDownloadCmd()
		dest := t.TempDir()
		downloadCmd.SetArgs([]string{"all", dest})
		require.NotPanics(t, func() { require.NoError(t, downloadCmd.Execute()) })

		require.FileExists(t, filepath.Join(dest, "app1.build", "output"))
		require.FileExists(t, filepath.Join(dest, "app1.build1", "output1"))
	})

	t.Run("downloadWithTaskID", func(t *testing.T) {
		initTest(t)

		downloadCmd := newReleaseDownloadCmd()
		dest := t.TempDir()
		downloadCmd.SetArgs([]string{"--tasks", "app1.build1", "all", dest})
		require.NotPanics(t, func() { require.NoError(t, downloadCmd.Execute()) })

		require.NoFileExists(t, filepath.Join(dest, "app1.build", "output"))
		require.FileExists(t, filepath.Join(dest, "app1.build1", "output1"))
	})

	t.Run("downloadWithNonExistingTaskID", func(t *testing.T) {
		initTest(t)
		downloadCmd := newReleaseDownloadCmd()
		downloadCmd.SetArgs([]string{"--tasks", "oooh.noes", "all", t.TempDir()})
		execCheck(t, downloadCmd, exitCodeError)
	})

	t.Run("downloadWithTaskIDWithoutOutputs", func(t *testing.T) {
		initTest(t)
		downloadCmd := newReleaseDownloadCmd()
		downloadCmd.SetArgs([]string{"--tasks", "app1.no-outputs", "all", t.TempDir()})
		execCheck(t, downloadCmd, exitCodeError)
	})
}
