package baur

import (
	"encoding/json"
	"fmt"
	"os"
)

type TaskInfoFile struct {
	TotalInputDigest string
	AppDir           string
	Outputs          []*taskInfoOutput
}

type taskInfoOutput struct {
	URI string
}

// CreateTempTaskInfofile creates a temporary file, writes the JSON encoding of
// c to it and returns the file path.
// The caller is responsible for deleting the file.
func (c *TaskInfoFile) ToTmpfile(tmpfileNamePart string) (string, error) {
	fd, err := os.CreateTemp("", "baur-taskinfo-"+tmpfileNamePart+".json")
	if err != nil {
		return "", fmt.Errorf("creating temporary file failed: %w", err)
	}

	err = json.NewEncoder(fd).Encode(c)
	if err != nil {
		return "", fmt.Errorf("encoding or writing task info as JSON to temporary file %q failed: %w", fd.Name(), err)
	}

	if err := fd.Close(); err != nil {
		return "", fmt.Errorf("closing temporary file %q failed: %w", fd.Name(), err)
	}

	return fd.Name(), nil
}
