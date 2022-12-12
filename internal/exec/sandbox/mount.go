package sandbox

import (
	"fmt"

	"golang.org/x/sys/unix"

	"github.com/simplesurance/baur/v3/internal/fs"
	"github.com/simplesurance/baur/v3/internal/log"
)

// MountAttrs specifies the attributes for filesystem mount and umount
// operations.
type MountAttrs struct {
	Source      string
	Target      string
	FsType      string
	MountFlags  uintptr
	UmountFlags int
	MountData   string
}

func (m *MountAttrs) String() string {
	return fmt.Sprintf(
		"fs: %q source: %q target: %q mount flags: %d mount options: %s, umount flags: %d",
		m.FsType, m.Source, m.Target, m.MountFlags, m.MountData, m.UmountFlags,
	)
}

type MountInfo interface {
	MountAttrs() *MountAttrs
}

func Mount(fs MountInfo) error {
	attrs := fs.MountAttrs()
	// TODO: improve this log message
	log.Debugf("mounting filesystem: %s", attrs)
	return unix.Mount(
		attrs.Source,
		attrs.Target,
		attrs.FsType,
		attrs.MountFlags,
		attrs.MountData,
	)
}

func Umount(fs MountInfo) error {
	attrs := fs.MountAttrs()
	log.Debugf("umounting %s filesystem at %s (umount flags: %d)", attrs.FsType, attrs.Target, attrs.UmountFlags)
	return unix.Unmount(attrs.Target, attrs.UmountFlags)
}

type OverlayFsMount struct {
	attrs MountAttrs
	// TODO: are the following fields actually needed?
	lowerDir string
	upperDir string
	workDir  string
}

func NewOverlayFSMount(lowerdir, upperdir, workdir, mntpoint string) *OverlayFsMount {
	return &OverlayFsMount{
		attrs: MountAttrs{
			Source: "none",
			Target: mntpoint,
			FsType: "overlay",
			MountData: fmt.Sprintf("userxattr,lowerdir=%s,upperdir=%s,workdir=%s",
				lowerdir, upperdir, workdir,
			),
		},
		lowerDir: lowerdir,
		upperDir: upperdir,
		workDir:  workdir,
	}
}

// Mkdirs creates the directories needed to mount the overlayFs.
// These are the lowerdir, upperdir, workdir and mntpoint directories.
// If the directories already exist, this is a noop.
func (f *OverlayFsMount) Mkdirs() error {
	return fs.Mkdirs(
		f.lowerDir,
		f.upperDir,
		f.workDir,
		f.attrs.Target,
	)

}

func (f *OverlayFsMount) MountAttrs() *MountAttrs {
	return &f.attrs
}
