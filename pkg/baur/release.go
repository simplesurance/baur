package baur

import (
	"encoding/json"
	"io"
)

type Release struct {
	ReleaseName  string
	Metadata     []byte `json:",omitempty"`
	Applications map[string]*ReleaseApp
}

type ReleaseApp struct {
	TaskRuns map[string]*ReleaseTaskRun
}

type ReleaseTaskRun struct {
	RunID   int
	Outputs map[int]*ReleaseOutput
}

type ReleaseOutput struct {
	Uploads []*ReleaseUpload
}

type ReleaseUpload struct {
	UploadMethod string
	URI          string
}

// ToJSON encodes the Release to JSON and writes the result to w.
// If excludeMetadata is true or no Metadata exists, the Metadata field is omitted from the result.
func (r *Release) ToJSON(w io.Writer, excludeMetadata bool) error {
	enc := json.NewEncoder(w)

	if !excludeMetadata {
		return enc.Encode(r)
	}

	cp := *r
	cp.Metadata = nil
	return enc.Encode(cp)
}

func (r *Release) WriteMetadata(w io.Writer) error {
	_, err := w.Write(r.Metadata)
	return err
}
