package cfg

import (
	"fmt"
	"os"

	"github.com/pelletier/go-toml"
)

// toFile marshals a struct to TOML format and writes it to a file.
func toFile(data interface{}, filepath string, overwrite bool) error {
	var openFlags int

	if overwrite {
		openFlags = os.O_WRONLY | os.O_CREATE | os.O_TRUNC
	} else {
		openFlags = os.O_WRONLY | os.O_CREATE | os.O_EXCL
	}

	f, err := os.OpenFile(filepath, openFlags, 0640)
	if err != nil {
		return err
	}

	encoder := toml.NewEncoder(f)
	encoder.Order(toml.OrderPreserve)
	err = encoder.Encode(data)
	if err != nil {
		f.Close()
		return err
	}

	err = f.Close()
	if err != nil {
		return fmt.Errorf("closing file failed: %w", err)
	}

	return err
}
