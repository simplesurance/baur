package gosource

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/simplesurance/baur/v3/internal/log"
	"github.com/simplesurance/baur/v3/internal/prettyprint"
	"github.com/simplesurance/baur/v3/internal/testutils/ostest"
	"github.com/simplesurance/baur/v3/internal/testutils/strtest"
)

type testCfg struct {
	Cfg struct {
		Environment []string
		Queries     []string
		BuildFlags  []string
		Tests       bool
		WorkingDir  string
	}
	ExpectedResults []string
}

func testCfgFromFile(t *testing.T, path string) *testCfg {
	var result testCfg
	fileContent, err := os.ReadFile(path)
	require.NoError(t, err)
	require.NoError(t, json.Unmarshal(fileContent, &result))

	return &result
}

func TestResolve(t *testing.T) {
	const testCfgFilename = "test_config.json"

	testdataDirs, err := filepath.Glob(filepath.Join("testdata", "*"))
	require.NoError(t, err)

	for i, d := range testdataDirs {
		testdataDirs[i], err = filepath.Abs(d)
		require.NoError(t, err)
	}

	for _, dir := range testdataDirs {
		t.Run(dir, func(t *testing.T) {
			log.RedirectToTestingLog(t)

			testCfg := testCfgFromFile(t, filepath.Join(dir, testCfgFilename))
			require.NotEmpty(t, testCfg.Cfg.WorkingDir)
			require.NotEmpty(t, testCfg.Cfg.Queries)
			require.NotEmpty(t, testCfg.ExpectedResults)

			ostest.Chdir(t, filepath.FromSlash(strings.ReplaceAll(testCfg.Cfg.WorkingDir, "$TESTDIR", dir)))

			for i := range testCfg.Cfg.Environment {
				testCfg.Cfg.Environment[i] = strings.ReplaceAll(testCfg.Cfg.Environment[i], "$TESTDIR", dir)
			}

			for i := range testCfg.ExpectedResults {
				testCfg.ExpectedResults[i] = strings.ReplaceAll(testCfg.ExpectedResults[i], "$TESTDIR", dir)
				// The path separators in the test config are Unix style "/", they need to be converted to "\" when running on Windows
				testCfg.ExpectedResults[i] = filepath.FromSlash(testCfg.ExpectedResults[i])
			}

			resolvedFiles, err := NewResolver(t.Logf).Resolve(
				context.Background(),
				dir,
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
