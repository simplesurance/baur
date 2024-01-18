// Package cfg implements the baur configuration file parser.
package cfg

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"os"

	"github.com/pelletier/go-toml"
)

type toFileOpts struct {
	overwrite bool
	commented bool
}

// toFileOpt is an option that can be passed to the ToFile functions
type toFileOpt func(*toFileOpts)

// ToFileOptOverwrite overwrite an existing file instead of returning an error
func ToFileOptOverwrite() toFileOpt { //nolint: revive // returns unexported type
	return func(o *toFileOpts) {
		o.overwrite = true
	}
}

// ToFileOptCommented comment every line in the config
func ToFileOptCommented() toFileOpt { //nolint: revive // returns unexported type
	return func(o *toFileOpts) {
		o.commented = true
	}
}

// toFile marshals a struct to TOML format and writes it to a file.
func toFile(data any, filepath string, opts ...toFileOpt) error {
	var buf bytes.Buffer
	var settings toFileOpts

	for _, opt := range opts {
		opt(&settings)
	}

	encoder := toml.NewEncoder(&buf)
	encoder.ArraysWithOneElementPerLine(true)
	encoder.Order(toml.OrderPreserve)

	err := encoder.Encode(data)
	if err != nil {
		return err
	}

	f, err := os.OpenFile(filepath, fileOpenFlags(settings.overwrite), 0640)
	if err != nil {
		return err
	}

	if settings.commented {
		if err := writeCommented(f, &buf); err != nil {
			f.Close()
			return err
		}
	} else {
		if _, err := io.Copy(f, &buf); err != nil {
			f.Close()
			return err
		}
	}

	err = f.Close()
	if err != nil {
		return fmt.Errorf("closing file failed: %w", err)
	}

	return err
}

func fileOpenFlags(overwrite bool) int {
	if overwrite {
		return os.O_WRONLY | os.O_CREATE | os.O_TRUNC
	}

	return os.O_WRONLY | os.O_CREATE | os.O_EXCL
}

func writeCommented(out io.Writer, in io.Reader) error {
	s := bufio.NewScanner(in)

	for s.Scan() {
		line := s.Text()

		if _, err := fmt.Fprintf(out, "# %s\n", line); err != nil {
			return err
		}
	}

	return s.Err()
}
