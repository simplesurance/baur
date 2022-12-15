package sandbox

import (
	"syscall"
)

type BindMount struct {
	attrs MountAttrs
}

func NewBindMount(src, mntpoint string) *BindMount {
	return &BindMount{
		attrs: MountAttrs{
			Source:     src,
			Target:     mntpoint,
			MountFlags: syscall.MS_BIND,
		},
	}
}

func (f *BindMount) MountAttrs() *MountAttrs {
	return &f.attrs
}
