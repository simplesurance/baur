package cfg

import (
	"fmt"

	"github.com/pelletier/go-toml"
)

// ReadReadVersion reads the config_version entry from a toml config file.
// If the key does not exist or is not a positive integer, an error is returned.
func ReadVersion(path string) (int, error) {
	const key = "config_version"

	cfg, err := toml.LoadFile(path)
	if err != nil {
		return -1, err
	}

	verI := cfg.Get("config_version")
	if verI == nil {
		return -1, fmt.Errorf("%q key does not exist in config", key)
	}

	ver, ok := verI.(int)
	if !ok {
		return -1, fmt.Errorf("%q value is '%v', expected an integer", key, ver)
	}

	if ver < 0 {
		return -1, fmt.Errorf("%q value is '%d', expected an positive integer", key, ver)
	}

	return ver, nil
}
