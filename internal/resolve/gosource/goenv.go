package gosource

import (
	"encoding/json"
	"fmt"
	"path/filepath"

	"github.com/simplesurance/baur/v1/internal/exec"
)

type goEnv struct {
	GoCache    string
	GoModCache string
	GoRoot     string
	GoPath     string
}

func getGoEnv(env []string) (*goEnv, error) {
	var result goEnv

	res, err := exec.Command("go", "env", "-json").Env(env).ExpectSuccess().Run()
	if err != nil {
		return nil, err
	}

	if err := json.Unmarshal(res.Output, &result); err != nil {
		return nil, fmt.Errorf("converting %q to json failed: %w", string(res.Output), err)
	}

	if result.GoModCache == "" {
		return nil, fmt.Errorf("go env returned an GOMODCACHE variable, the variable must be set")
	}

	if result.GoRoot == "" {
		return nil, fmt.Errorf("go env returned an GOROOT variable, the variable must be set")
	}

	// the variables can contain e.g. trailing directory seperators, when
	// they were set manually to such a value, to ensure this does not
	// cause issues when using them for path replacements later, clean all
	// paths
	result.GoModCache = filepath.Clean(result.GoModCache)
	result.GoRoot = filepath.Clean(result.GoRoot)
	result.GoPath = filepath.Clean(result.GoPath)
	result.GoCache = filepath.Clean(result.GoCache)

	return &result, nil
}
