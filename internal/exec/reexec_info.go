package exec

import (
	"bytes"
	"encoding/gob"
	"io"
)

// TODO: This is the wrong place for this structure.
//  It not required by the other exec commands.
// This should be in an internal package that is accessible by the command implementation and the taskrunner.

type ReExecInfo struct {
	RepositoryDir       string
	OverlayFsTmpDir     string
	Command             []string
	WorkingDirectory    string
	AllowedFilesRelPath []string
}

func (s *ReExecInfo) Encode() (*bytes.Buffer, error) {
	var buf bytes.Buffer

	err := gob.NewEncoder(&buf).Encode(s)
	if err != nil {
		return nil, err
	}

	return &buf, nil
}

func ReExecInfoDecode(r io.ReadCloser) (*ReExecInfo, error) {
	var result ReExecInfo

	err := gob.NewDecoder(r).Decode(&result)
	if err != nil {
		return nil, err
	}

	return &result, err
}
