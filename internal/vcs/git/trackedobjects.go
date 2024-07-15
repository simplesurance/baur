package git

import (
	"context"
	"errors"
	"path/filepath"
	"sync"

	"github.com/simplesurance/baur/v5/internal/set"
)

var ErrObjectNotFound = errors.New("git object id not found, file might not exist, untracked or modified")

type PrintfFn func(format string, a ...any)

type TrackedObject struct {
	ObjectID string
	Mode     Mode
}

// TrackedObjects stores information about tracked and unmodified files in a
// git repository.
type TrackedObjects struct {
	info       map[string]*TrackedObject
	loadDbOnce sync.Once
	loadErr    error

	repositoryDir string
	debugLogf     PrintfFn
}

func NewTrackedObjects(gitRepositoryDir string, debugLogf PrintfFn) *TrackedObjects {
	return &TrackedObjects{
		repositoryDir: gitRepositoryDir,
		debugLogf:     debugLogf,
		info:          map[string]*TrackedObject{},
	}
}

func (h *TrackedObjects) loadFromRepositoryOnce(ctx context.Context) error {
	h.loadDbOnce.Do(func() { h.loadErr = h.loadFromRepository(ctx) })
	return h.loadErr
}

func (h *TrackedObjects) loadFromRepository(ctx context.Context) error {
	ch := make(chan *Object)
	finishedCh := make(chan struct{})
	go h.createDb(ch, finishedCh)

	err := LsFiles(ctx, h.repositoryDir, true, ch)
	if err != nil {
		return err
	}

	<-finishedCh

	h.debugLogf("gitobjectid: loaded %d object IDs from git repository (%s)\n", len(h.info), h.repositoryDir)

	return nil
}

func (h *TrackedObjects) createDb(ch <-chan *Object, finishedCh chan struct{}) {
	const objectTypeFileOrSymlink = ObjectTypeFile | ObjectTypeSymlink

	m := map[string]*TrackedObject{}
	outdated := set.Set[string]{}

	defer close(finishedCh)

	for obj := range ch {
		if obj.Mode&objectTypeFileOrSymlink == 0 {
			continue
		}

		absPath := filepath.Join(h.repositoryDir, obj.RelPath)
		if obj.Status == ObjectStatusCached {
			if outdated.Contains(absPath) {
				continue
			}
			m[absPath] = &TrackedObject{
				ObjectID: obj.Name,
				Mode:     obj.Mode,
			}

			continue
		}

		outdated.Add(absPath)
		delete(m, absPath)
	}

	h.info = m
}

// Get returns a TrackedObject for the file at absPath in the git repository.
// If the file does not exist, is untracked or modified ErrObjectNotFound is returned.
//
// On the first call, the objects are read from the Git repository. If it
// fails, the call and following runs are returning the occurred error.
func (h *TrackedObjects) Get(ctx context.Context, absPath string) (*TrackedObject, error) {
	if err := h.loadFromRepositoryOnce(ctx); err != nil {
		return nil, err
	}

	o, exists := h.info[absPath]
	if !exists {
		return nil, ErrObjectNotFound
	}

	return o, nil
}
