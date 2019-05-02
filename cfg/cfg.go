package cfg

import (
	"os"

	"github.com/pelletier/go-toml"
	"github.com/pkg/errors"
)

// toFile serializes a struct to TOML format and writes it to a file.
func toFile(data interface{}, filepath string, overwrite bool) error {
	var openFlags int

	tomlData, err := toml.Marshal(data)
	if err != nil {
		return errors.Wrapf(err, "marshalling failed")
	}

	if overwrite {
		openFlags = os.O_WRONLY | os.O_CREATE | os.O_TRUNC
	} else {
		openFlags = os.O_WRONLY | os.O_CREATE | os.O_EXCL
	}

	f, err := os.OpenFile(filepath, openFlags, 0640)
	if err != nil {
		return err
	}

	_, err = f.Write(tomlData)
	if err != nil {
		return errors.Wrap(err, "writing to file failed")
	}

	err = f.Close()
	if err != nil {
		return errors.Wrap(err, "closing file failed")
	}

	return err
}
