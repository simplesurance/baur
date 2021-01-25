package gosource

import (
	"encoding/json"
	"fmt"
	"path/filepath"

	"github.com/simplesurance/baur/v1/internal/exec"
)

type goEnv struct {
	// GoCache is empty when the GOCACHE value is "off"
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

	if result.GoCache == "off" {
		result.GoCache = ""
	}

	// Which are valid paths for the go environment variables differs.
	// - If GOCACHE is set to a relative path, "go env" returns "off" as
	//   value,
	// - GOROOT and GOMODCACHE can be set to relative paths,
	// - If GOPATH is set to a relative path "go env" fails with an error
	// Unclean paths can also be assigned (e.g. trailing slashes).
	// To have consistent paths, filepath.Abs() is run for each of them.
	if result.GoModCache, err = filepath.Abs(result.GoModCache); err != nil {
		return nil, fmt.Errorf("GOMODCACHE: %w", err)
	}

	if result.GoRoot, err = filepath.Abs(result.GoRoot); err != nil {
		return nil, fmt.Errorf("GOROOT: %w", err)
	}

	if result.GoPath, err = filepath.Abs(result.GoPath); err != nil {
		return nil, fmt.Errorf("GOPATH: %w", err)
	}

	if result.GoCache != "" {
		if result.GoCache, err = filepath.Abs(result.GoCache); err != nil {
			return nil, fmt.Errorf("GOCACHE: %w", err)
		}
	}

	return &result, nil
}
