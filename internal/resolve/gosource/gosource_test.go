package gosource

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/simplesurance/baur/v2/internal/log"
	"github.com/simplesurance/baur/v2/internal/prettyprint"
	"github.com/simplesurance/baur/v2/internal/testutils/strtest"
)

func TestResolve(t *testing.T) {
	const testCfgFilename = "test_config.json"

	type testCfg struct {
		Cfg struct {
			Environment []string
			Queries     []string
			BuildFlags  []string
			Tests       bool
		}
		ExpectedResults []string
	}

	testdataDirs, err := filepath.Glob(filepath.Join("testdata", "*"))
	require.NoError(t, err)

	for i, d := range testdataDirs {
		testdataDirs[i], err = filepath.Abs(d)
		require.NoError(t, err)
	}

	for _, dir := range testdataDirs {
		t.Run(dir, func(t *testing.T) {
			var testCfg testCfg

			log.StdLogger.SetOutput(log.NewTestLogOutput(t))

			require.NoError(t, os.Chdir(dir))

			fileContent, err := os.ReadFile(testCfgFilename)
			require.NoError(t, err)

			require.NoError(t, err, json.Unmarshal(fileContent, &testCfg))

			cwd, err := os.Getwd()
			require.NoError(t, err)

			for i := range testCfg.Cfg.Environment {
				testCfg.Cfg.Environment[i] = strings.Replace(testCfg.Cfg.Environment[i], "$WORKDIR", cwd, -1)
			}

			for i := range testCfg.ExpectedResults {
				testCfg.ExpectedResults[i] = strings.Replace(testCfg.ExpectedResults[i], "$WORKDIR", cwd, -1)
				// The path separators in the test config are Unix style "/", they need to be converted to "\" when running on Windows
				testCfg.ExpectedResults[i] = filepath.FromSlash(testCfg.ExpectedResults[i])
			}

			resolvedFiles, err := NewResolver(t.Logf).Resolve(
				context.Background(),
				cwd,
				testCfg.Cfg.Environment,
				testCfg.Cfg.BuildFlags,
				testCfg.Cfg.Tests,
				testCfg.Cfg.Queries,
			)
			require.NoError(t, err)

			t.Logf("gosources resolved to: %s", prettyprint.AsString(resolvedFiles))

			for _, path := range resolvedFiles {
				if !strtest.InSlice(testCfg.ExpectedResults, path) {
					t.Errorf("resolved file contain %q but it's not part of the ExpectedResult slice: %+v", path, testCfg.ExpectedResults)
				}
			}

			for _, path := range testCfg.ExpectedResults {
				if !strtest.InSlice(resolvedFiles, path) {
					t.Errorf("resolved go source is missing %q in %+v", path, resolvedFiles)
				}
			}
		})
	}
}
