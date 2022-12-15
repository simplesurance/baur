package sandbox

import (
	"fmt"
	"syscall"

	"golang.org/x/sys/unix"

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

	if log.DebugEnabled() {
		var mountType string
		if attrs.FsType != "" {
			mountType = attrs.FsType + " filesystem"
		} else if (attrs.MountFlags & syscall.MS_BIND) != 0 {
			mountType = "bind mount"

		}

		log.Debugf("umounting %s at %s (umount flags: %d)", mountType, attrs.Target, attrs.UmountFlags)
	}

	return unix.Unmount(attrs.Target, attrs.UmountFlags)
}
