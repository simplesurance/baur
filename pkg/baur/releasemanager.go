package baur

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/simplesurance/baur/v3/internal/fs"
	"github.com/simplesurance/baur/v3/internal/output/s3"
	"github.com/simplesurance/baur/v3/internal/set"
	"github.com/simplesurance/baur/v3/pkg/storage"
)

type S3Downloader interface {
	Download(ctx context.Context, bucket, key, filepath string) error
}

type ReleaseManager struct {
	storage storage.Storer
	logger  Logger
	s3clt   S3Downloader
}

func NewReleaseManager(storage *storage.Storer, s3clt *s3.Client, logger Logger) *ReleaseManager {
	return &ReleaseManager{
		storage: *storage,
		s3clt:   s3clt,
		logger:  logger,
	}
}

type DownloadOutputsParams struct {
	ReleaseName string
	DestDir     string
	// TaskIDs specifies the tasks for which the output are downloaded, if
	// empty outputs of all tasks are downloaded.
	TaskIDs []string
	// DownloadStartFn is a callback function that is called before the
	// download of an output starts
	DownloadStartFn func(taskID, url, destFilepath string)
}

// DownloadOutputs downloads outputs of a release.
// Currently it is only supported to download outputs that have been uploaded
// to S3, others are ignored.
// If the same output was uploaded to multiple S3 destinations, it is only
// downloaded from one of them.
// If a task specified in [DownloadOutputs.TaskIDs] is not part of the release,
// an error is returned.
func (m *ReleaseManager) DownloadOutputs(
	ctx context.Context,
	p DownloadOutputsParams,
) error {
	tasksRuns, err := m.storage.ReleaseTaskRuns(ctx, p.ReleaseName)
	if err != nil {
		return err
	}

	// downloadedOutputs keeps track of which outputs per task has been downloaded.
	// Each output can be uploaded to multiple destinations, to prevent
	// downloading the same output multiple times, only download the first
	// upload to S3 per output.
	downloadedOutputs := set.Set[string]{}

	includes := set.From(p.TaskIDs)
	if err := validateWantedTaskIDsHaveOutputs(tasksRuns, includes); err != nil {
		return err
	}

	for _, tr := range tasksRuns {
		tid := taskID(tr.AppName, tr.TaskName)
		if len(includes) != 0 && !includes.Contains(tid) {
			continue
		}

		if tr.OutputName == "" {
			m.logger.Debugf(
				"%s: skipping task run, it did not produce any outputs\n",
				tid, storage.UploadMethodS3,
			)
			continue
		}

		if tr.UploadMethod != storage.UploadMethodS3 {
			m.logger.Debugf(
				"%s: %s: %s: skipping download, upload method is %s, only %s is supported\n",
				tid,
				tr.OutputName,
				tr.URI,
				tr.UploadMethod,
				storage.UploadMethodS3,
			)
			continue
		}

		taskOutputID := tid + "." + strconv.Itoa(tr.OutputID)
		if downloadedOutputs.Contains(taskOutputID) {
			m.logger.Debugf(
				"%s: %s: skipping download, output already has been downloaded from another destination \n",
				tid,
				tr.OutputName,
			)
			continue
		}

		destDir := filepath.Join(p.DestDir, tid)
		if err := fs.Mkdir(destDir); err != nil {
			return err
		}
		destFile := filepath.Join(destDir, filepath.Base(tr.OutputName))

		bucket, objectKey, err := s3.ParseURL(tr.URI)
		if err != nil {
			return fmt.Errorf(
				"%s (%d): stored uri of output %s is malformed: %s: %w",
				tid, tr.RunID, tr.OutputName, tr.URI, err,
			)
		}

		if p.DownloadStartFn != nil {
			p.DownloadStartFn(tid, tr.URI, destFile)
		}

		err = m.s3clt.Download(ctx, bucket, objectKey, destFile)
		if err != nil {
			return fmt.Errorf(
				"%s (%d): download of %s failed: %w",
				tid, tr.RunID, tr.URI, err,
			)
		}

		downloadedOutputs.Add(taskOutputID)
	}

	return nil
}

// validateWantedTaskIDsHaveOutputs ensures that taskRuns contains a task run
// with an S3 upload for each task ID in taskIDs.
// If it doesn't, an error is returned.
// if taskIDs is empty, nil is returned.
func validateWantedTaskIDsHaveOutputs(
	taskRuns []*storage.ReleaseTaskRunsResult,
	taskIDs set.Set[string],
) error {
	var missingTaskIDs []string
	var tasksMissingS3Uploads []string

	if len(taskIDs) == 0 {
		return nil
	}

	releaseTaskIDs := map[string][]*storage.ReleaseTaskRunsResult{}
	for _, tr := range taskRuns {
		tid := taskID(tr.AppName, tr.TaskName)
		taskRuns, exists := releaseTaskIDs[tid]
		if !exists {
			releaseTaskIDs[tid] = []*storage.ReleaseTaskRunsResult{tr}
			continue
		}
		releaseTaskIDs[tid] = append(taskRuns, tr)
	}

	for wantedTaskID := range taskIDs {
		taskRuns, exists := releaseTaskIDs[wantedTaskID]
		if !exists {
			missingTaskIDs = append(missingTaskIDs, wantedTaskID)
			continue
		}

		found := false
		for _, tr := range taskRuns {
			if tr.UploadMethod != storage.UploadMethodS3 {
				continue
			}
			found = true
			break
		}

		if !found {
			tasksMissingS3Uploads = append(tasksMissingS3Uploads, wantedTaskID)
		}
	}

	if len(missingTaskIDs) == 0 && len(tasksMissingS3Uploads) == 0 {
		return nil
	}

	var errStr string
	if len(missingTaskIDs) > 0 {
		errStr = "The following task IDs are not part of the release: " + strings.Join(missingTaskIDs, ",")
	}
	if len(tasksMissingS3Uploads) > 0 {
		if errStr != "" {
			errStr += "\n"
		}
		errStr += "The following tasks do not have an S3 output: " + strings.Join(tasksMissingS3Uploads, ",")
	}

	return errors.New(errStr)
}
