package sandbox

// TODO: move this package out of exec? It's not related, is it?

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"syscall"
	"time"

	"github.com/simplesurance/baur/v3/internal/fs"
	"github.com/simplesurance/baur/v3/internal/log"
)

// ReExecDataPipeFD is the id of the file-descriptor to which ReExecInNs()
// pipes data to the new process.
const ReExecDataPipeFD uintptr = 3

// ReExecInNs executes the current running binary (/proc/self/exe) again in
// a new Linux user- and mount Namespace.
// args are passed as command line arguments.
// data is piped to the the process via file-descriptor 3. If data is passed,
// the process must read it, otherwise the operation will fail.
// data is usually information that tells the new process what to do.
// The reexecuted process will run with uid & gid 0 but have the same
// permissions then the currently executing process.
func ReExecInNs(ctx context.Context, args []string, data io.Reader) error {
	pipeReader, pipeWriter, err := os.Pipe()
	if err != nil {
		return err
	}

	cmd := exec.CommandContext(ctx, "/proc/self/exe", args...)
	cmd.Stderr = os.Stderr
	cmd.Stdout = os.Stdout
	cmd.Stdin = os.Stdin
	cmd.ExtraFiles = []*os.File{pipeReader}
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Pdeathsig:    syscall.SIGKILL,
		Unshareflags: syscall.CLONE_NEWUSER | syscall.CLONE_NEWNS,
		UidMappings: []syscall.SysProcIDMap{
			{
				ContainerID: 0,
				HostID:      os.Getuid(),
				Size:        1,
			},
		},
		GidMappings: []syscall.SysProcIDMap{
			{
				ContainerID: 0,
				HostID:      os.Getgid(),
				Size:        1,
			},
		},
	}

	defer pipeReader.Close()

	log.Debugf("starting new process of myself (%q) in new user and mount namespaces", cmd)

	// lock to thread because of:
	// https://github.com/golang/go/issues/27505#issuecomment-713706104
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	err = cmd.Start()
	if err != nil {
		pipeWriter.Close()
		return err
	}

	err = pipeWriter.SetWriteDeadline(time.Now().Add(30 * time.Second))
	if err != nil {
		log.Debugf("WARN: setting deadling on pipe failed: %s", err)
	}

	_, err = io.Copy(pipeWriter, data)
	if err != nil {
		err = fmt.Errorf("piping data to child process failed: %w", err)
		killErr := cmd.Process.Kill()
		if killErr != nil {
			return fmt.Errorf("%s, killing the chil process (pid: %d) too: %s",
				err, cmd.Process.Pid, killErr)
		}

		return err
	}
	_ = pipeWriter.Close()

	return cmd.Wait()
}

// HiddenDirectory is directory in which all files that are not in an allow
// list are hidden.
type HiddenDirectory struct {
	dir       string
	tmpdir    string
	overlayFs *OverlayFsMount
	bindMount *BindMount
}

// HideFiles hides all files in dir that are not listen in unhiddenFiles.
// unhiddenFiles must only contains paths that are a subpath of dir.
// tmpdir is a path to a directory that is created and used to store temporary
// overlayFS directories. tmpdir must not exist, otherwise an error is
// returned.
//
// The files are hidden by using an OverlayFs and BindFs.
// The directory is mounted as an overlayFS and all files that are not in the
// allowedList are deleted in the mounted overlayFS.
// The overlayFS is then bind mounted on top of the HiddenDirectory.
// When the directory is not needed anymore, Close() should be called to umount
// the filesystems and remove the temporary files.
// tmpdir must be on a filesystem that is **not** mounted with the nodev option.
// OverlayFs uses character device files to represent deleted dirs and files.
func HideFiles(dir, tmpdir string, unhiddenFiles []string) (*HiddenDirectory, error) {
	hd := HiddenDirectory{
		dir:    dir,
		tmpdir: tmpdir,
	}

	if _, err := os.Stat(tmpdir); err == nil {
		return nil, fmt.Errorf("tmpdir %q must not exist", tmpdir)
	}

	overlayFs := NewOverlayFSMount(
		dir,
		filepath.Join(tmpdir, "upper"),
		filepath.Join(tmpdir, "work"),
		filepath.Join(tmpdir, "mnt"),
	)
	err := overlayFs.Mkdirs()
	if err != nil {
		if rmErr := os.RemoveAll(tmpdir); rmErr != nil {
			return nil, fmt.Errorf("%s, deleting tmpdir during cleanup also failed: %s",
				err, rmErr)
		}

		return nil, err
	}
	if err := Mount(overlayFs); err != nil {
		err = fmt.Errorf("mounting overlayfs failed: %w", err)
		if cleanupErr := hd.Close(); cleanupErr != nil {
			return nil, fmt.Errorf("%s, cleanup failed too: %s", err, cleanupErr)
		}

		return nil, err
	}
	hd.overlayFs = overlayFs

	err = fs.RemoveAllExcept(overlayFs.MountPoint(), unhiddenFiles)
	if err != nil {
		err = fmt.Errorf("removing non-input files from overlayfs failed: %w", err)

		if cleanupErr := hd.Close(); cleanupErr != nil {
			return nil, fmt.Errorf("%s, cleanup failed too: %s", err, cleanupErr)
		}

		return nil, err
	}

	bindMount := NewBindMount(hd.overlayFs.MountPoint(), dir)
	if err := Mount(bindMount); err != nil {
		err = fmt.Errorf("bind mounting %q to %q failed: %w", hd.overlayFs.MountPoint(), dir, err)
		if cleanupErr := hd.Close(); cleanupErr != nil {
			return nil, fmt.Errorf("%s, cleanup failed too: %s", err, cleanupErr)
		}

		return nil, err
	}
	hd.bindMount = bindMount

	return &hd, nil
}

func (h *HiddenDirectory) Close() error {
	if h.bindMount != nil {
		if err := Umount(h.bindMount); err != nil {
			return fmt.Errorf("umounting bind mount: %w", err)
		}
	}

	if h.overlayFs != nil {
		if err := Umount(h.overlayFs); err != nil {
			return fmt.Errorf("umounting overlayfs failed: %w", err)
		}
	}

	if err := os.RemoveAll(h.tmpdir); err != nil {
		return fmt.Errorf("deleting tmpdir failed: %w", err)
	}

	return nil
}
