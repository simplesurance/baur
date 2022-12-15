package sandbox

import (
	"fmt"

	"github.com/simplesurance/baur/v3/internal/fs"
)

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

func (f *OverlayFsMount) MountPoint() string {
	return f.attrs.Target
}
