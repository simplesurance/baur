package baur

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"

	"github.com/simplesurance/baur/v3/pkg/storage"
)

type Release struct {
	Name string
	// Metadata is lazily assigned when [Release.ToJSON] is called.
	Metadata []byte `json:",omitempty"`
	// MetadataReader can be read to retrieve the stored Metadata. The
	// reader is responsible for seeking it to the start before reading.
	MetadataReader io.ReadSeeker `json:"-"`
	TaskRuns       []*ReleaseTaskRun
}

type ReleaseTaskRun struct {
	ID              int
	ApplicationName string
	TaskName        string
	Outputs         []*ReleaseOutput
}

type ReleaseOutput struct {
	URI string
}

// ReleaseFromStorage retrieves information about the release named releaseName
// from clt.
// The metadata can be read from [Release.Metadata], [Release.Metadata] is only
// populated when [Release.ToJSON] is called.
func ReleaseFromStorage(ctx context.Context, clt storage.Storer, releaseName string) (*Release, error) {
	release, err := clt.Release(ctx, releaseName)
	if err != nil {
		return nil, err
	}

	taskRuns := make([]*ReleaseTaskRun, 0, len(release.TaskRunIDs))
	for _, runID := range release.TaskRunIDs {
		// TODO: optimize the storage queries to only return the
		// information that we use here
		storageTr, err := clt.TaskRun(ctx, runID)
		if err != nil {
			return nil, err
		}

		tr := ReleaseTaskRun{
			ID:              storageTr.ID,
			ApplicationName: storageTr.ApplicationName,
			TaskName:        storageTr.TaskName,
		}

		outputs, err := clt.Outputs(ctx, storageTr.ID)
		if err != nil {
			if errors.Is(err, storage.ErrNotExist) {
				taskRuns = append(taskRuns, &tr)
				continue
			}
			return nil, err
		}

		for _, output := range outputs {
			tr.Outputs = make([]*ReleaseOutput, 0, len(output.Uploads))
			for _, upload := range output.Uploads {
				tr.Outputs = append(tr.Outputs, &ReleaseOutput{URI: upload.URI})
			}
		}
		taskRuns = append(taskRuns, &tr)
	}

	return &Release{
		Name:           releaseName,
		MetadataReader: release.Metadata,
		TaskRuns:       taskRuns,
	}, nil

}

// ToJSON encodes the Release to JSON and writes the result to w.
// If excludeMetadata is true or no Metadata exists, the Metadata field is omitted from the result.
func (r *Release) ToJSON(w io.Writer, excludeMetadata bool) error {
	r.Metadata = nil
	if !excludeMetadata && r.MetadataReader != nil {
		_, err := r.MetadataReader.Seek(0, io.SeekStart)
		if err != nil {
			return fmt.Errorf("seeking metadata reader failed: %w", err)
		}

		metadata, err := io.ReadAll(r.MetadataReader)
		if err != nil {
			return fmt.Errorf("reading metadata failed: %w", err)
		}
		r.Metadata = metadata
	}

	enc := json.NewEncoder(w)
	enc.SetIndent("", "      ")
	return enc.Encode(r)
}

func (r *Release) WriteMetadata(w io.Writer) error {
	_, err := r.MetadataReader.Seek(0, io.SeekStart)
	if err != nil {
		return fmt.Errorf("seeking metadata reader failed: %w", err)
	}
	_, err = io.Copy(w, r.MetadataReader)
	return err
}
