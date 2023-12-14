package gitobjectid

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"sync"

	"github.com/simplesurance/baur/v3/internal/digest"
	"github.com/simplesurance/baur/v3/internal/fs"
	"github.com/simplesurance/baur/v3/internal/vcs/git"
)

type PrintfFn func(format string, a ...any)

// Calc calculates git object IDs for files.
// Object IDs are read the first time a digest is requested from the git repository.
type Calc struct {
	objectIDs    map[string]string
	symlinkPaths map[string]struct{}
	loadDbOnce   sync.Once
	loadErr      error

	repositoryDir string
	debugLogf     PrintfFn
}

func New(gitRepositoryDir string, debugLogf PrintfFn) *Calc {
	return &Calc{
		repositoryDir: gitRepositoryDir,
		debugLogf:     debugLogf,
		objectIDs:     map[string]string{},
		symlinkPaths:  map[string]struct{}{},
	}
}

func (h *Calc) loadFromRepositoryOnce(ctx context.Context) error {
	h.loadDbOnce.Do(func() { h.loadErr = h.loadFromRepository(ctx) })
	return h.loadErr
}

func (h *Calc) loadFromRepository(ctx context.Context) error {
	ch := make(chan *git.Object)
	finishedCh := make(chan struct{})
	go h.createDb(ch, finishedCh)

	err := git.LsFiles(ctx, h.repositoryDir, true, ch)
	if err != nil {
		return err
	}

	<-finishedCh

	h.debugLogf("gitobjectid: loaded %d object IDs from git repository (%s)\n", len(h.objectIDs), h.repositoryDir)

	return nil
}

func (h *Calc) createDb(ch <-chan *git.Object, finishedCh chan struct{}) {
	m := map[string]string{}
	outdated := map[string]struct{}{}

	defer close(finishedCh)

	for obj := range ch {
		if obj.IsSymlink() {
			h.symlinkPaths[filepath.Join(h.repositoryDir, obj.RelPath)] = struct{}{}
			continue
		}
		if !obj.IsFile() {
			continue
		}

		absPath := filepath.Join(h.repositoryDir, obj.RelPath)
		if obj.Status == git.ObjectStatusCached {
			if _, exists := outdated[absPath]; exists {
				continue
			}
			m[absPath] = obj.Name
			continue
		}

		outdated[absPath] = struct{}{}
		delete(m, absPath)
	}

	h.objectIDs = m
}

// File returns a git object ID as digest for the the file at absPath.
//
// On the first call of this function, the git object IDs are loaded from
// the git repository.
func (h *Calc) File(absPath string) (*digest.Digest, error) {
	// if the ID does not exist in the cache, it is calculated.
	// The calculated ID could be added to the cache afterwards to be available for further loookups.
	// This is not done because baur already has the
	// InputFileSingletonCache on top, which prevents multiple digest
	// calculations for the same path.
	if !filepath.IsAbs(absPath) {
		return nil, errors.New("file path is not absolute")
	}

	if err := h.loadFromRepositoryOnce(context.TODO()); err != nil {
		return nil, fmt.Errorf("loading git object ids failed: %w", err)
	}

	objectID, exists := h.objectIDs[absPath]
	if exists {
		return digest.FromStrDigest(objectID, digest.GitObjectID)
	}

	if _, isSymlink := h.symlinkPaths[absPath]; !isSymlink {
		d, err := h.fileWithoutCache(absPath)
		if err != nil {
			return nil, fmt.Errorf("git object id does not exist in cache and could not be calculated: %w", err)
		}
		return d, nil
	}

	// if it is a known symlink, the db might contain the entry for the target path
	p, err := fs.RealPath(absPath)
	if err != nil {
		return nil, err
	}
	if objectID, exists = h.objectIDs[p]; exists {
		return digest.FromStrDigest(objectID, digest.GitObjectID)
	}

	d, err := h.fileWithoutCache(absPath)
	if err != nil {
		return nil, fmt.Errorf("git object id does not exist in cache and could not be calculated: %w", err)
	}
	return d, nil
}

func (h *Calc) fileWithoutCache(absPath string) (*digest.Digest, error) {
	h.debugLogf("gitobjectid: object id not in cache, calculating ID for %q", absPath)
	objectID, err := git.ObjectID(context.TODO(), absPath, h.repositoryDir)
	if err != nil {
		return nil, err
	}

	return digest.FromStrDigest(objectID, digest.GitObjectID)
}
