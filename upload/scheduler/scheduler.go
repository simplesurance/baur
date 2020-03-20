package scheduler

import "time"

// Manager is an interface for upload managers
type Manager interface {
	Add(Job)
	Start()
	Stop()
}

// Result result of an upload attempt
type Result struct {
	Err   error
	URL   string
	Start time.Time
	Stop  time.Time
	Job   Job
}
