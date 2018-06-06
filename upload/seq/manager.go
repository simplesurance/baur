// Package seq implements a simple Sequential Uploader. Upload jobs are
// processed sequentially in-order.
package seq

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/simplesurance/baur/log"
	"github.com/simplesurance/baur/upload"
)

// Uploader is a sequential uploader
type Uploader struct {
	s3             upload.S3Uploader
	docker         upload.DockerUploader
	lock           sync.Mutex
	queue          []upload.Job
	stopProcessing bool
	statusChan     chan<- *upload.Result
}

// New initializes a sequential uploader
// Status chan must have a buffer count > 1 otherwise a deadlock occurs
func New(s3Uploader upload.S3Uploader, dockerUploader upload.DockerUploader, status chan<- *upload.Result) *Uploader {
	return &Uploader{
		s3:         s3Uploader,
		statusChan: status,
		lock:       sync.Mutex{},
		queue:      []upload.Job{},
		docker:     dockerUploader,
	}
}

// Add adds a new upload job, can be called after Start()
func (u *Uploader) Add(job upload.Job) {
	u.lock.Lock()
	defer u.lock.Unlock()

	u.queue = append(u.queue, job)
}

// Start starts uploading jobs in the queue.
// If the statusChan buffer is full, uploading will be blocked.
func (u *Uploader) Start() {
	for {
		var job upload.Job

		u.lock.Lock()
		if len(u.queue) > 0 {
			job = u.queue[0]
			u.queue = u.queue[1:]
		}
		u.lock.Unlock()

		if job != nil {
			var err error
			var url string
			startTs := time.Now()

			log.Debugf("uploading %s\n", job)
			if job.Type() == upload.JobS3 {
				url, err = u.s3.Upload(job.LocalPath(), job.RemoteDest())
			} else if job.Type() == upload.JobDocker {
				url, err = u.docker.Upload(context.Background(), job.LocalPath(), job.RemoteDest())
			} else {
				panic(fmt.Sprintf("invalid job %+v", job))
			}

			u.statusChan <- &upload.Result{
				Err:      err,
				URL:      url,
				Duration: time.Since(startTs),
				Job:      job,
			}
		}

		u.lock.Lock()
		if len(u.queue) == 0 {
			time.Sleep(time.Second)
		}

		if u.stopProcessing {
			close(u.statusChan)
			u.lock.Unlock()
			return
		}
		u.lock.Unlock()

	}
}

// Stop stops the uploader
func (u *Uploader) Stop() {
	u.lock.Lock()
	u.stopProcessing = true
	u.lock.Unlock()
}
