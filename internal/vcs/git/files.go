package git

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/simplesurance/baur/v3/internal/exec"
)

const (
	// source of the descriptions: git-ls-files manpage (Git 2.43.0)

	// H tracked file that is not either unmerged or skip-worktree
	ObjectStatusCached byte = 'H'
	// S tracked file that is skip-worktree
	ObjectStatusSkipWorktree byte = 'S'
	// M tracked file that is unmerged
	ObjectStatusUnmerged byte = 'M'
	// R tracked file with unstaged removal/deletion
	ObjectStatusRemoved byte = 'R'
	// C tracked file with unstaged modification/change
	ObjectStatusChanged byte = 'C'
	// K untracked paths which are part of file/directory conflicts which
	// prevent checking out tracked files
	ObjectStatusToBeKilled byte = 'K'
	// ? untracked file
	ObjectStatusUntracked byte = '?'
	// U file with resolve-undo information
	ObjectStatusWithResolveUndoInfo byte = 'U'
)

// Mode is the file mode from git (https://git-scm.com/book/en/v2/Git-Internals-Git-Objects)
// It is either:
// 100644 - normal file,
// 100755 - executable file,
// 120000 symbolic link,
type Mode uint32

const (
	ObjectTypeSymlink Mode = 0120000
	ObjectTypeFile    Mode = 0100000
)

type Object struct {
	Status byte
	Mode   Mode
	// Name is the hash of the object
	Name    string
	RelPath string
}

func (o *Object) String() string {
	return fmt.Sprintf("status: %c mode: %o name: %s path: %s", o.Status, o.Mode, o.Name, o.RelPath)
}

func (o *Object) IsSymlink() bool {
	return o.Mode&ObjectTypeSymlink == ObjectTypeSymlink
}

func (o *Object) IsFile() bool {
	return o.Mode&ObjectTypeFile == ObjectTypeFile
}

func scanNullTerminatedLines(data []byte, atEOF bool) (advance int, token []byte, err error) {
	if atEOF && len(data) == 0 {
		return 0, nil, nil
	}
	if i := bytes.IndexByte(data, '\x00'); i >= 0 {
		return i + 1, data[0:i], nil
	}
	if atEOF {
		return len(data), data, nil
	}
	return 0, nil, nil
}

// LsFiles retrieves information about files in the index and working tree.
// The information is sent to ch.
// For a path multiple objects might be sent, if the status in the index and working tree differs.
// Listing files in git submodules is not supported.
// The function closes ch when it terminates.
func LsFiles(ctx context.Context, dir string, skipUntracked bool, ch chan<- *Object) error {
	defer close(ch)
	reader, writer := io.Pipe()

	parseCh := make(chan error)
	go func() {
		err := parseLsFilesLines(reader, skipUntracked, ch)
		reader.CloseWithError(err)
		parseCh <- err
		close(parseCh)
	}()

	_, err := exec.Command(
		"git", "ls-files",
		"--cached", "--modified", "--other",
		"--exclude-standard",
		"--full-name",
		"-t",
		"--stage",
		"-z",
	).
		Directory(dir).
		ExpectSuccess().
		Stdout(writer).
		Run(ctx)
	_ = writer.Close()
	parseErr := <-parseCh
	return errors.Join(err, parseErr)
}

func parseLsFilesLines(reader io.Reader, skipUntracked bool, ch chan<- *Object) error {
	sc := bufio.NewScanner(reader)
	sc.Split(scanNullTerminatedLines)

	for sc.Scan() {
		var o Object
		// example line: H 100644 53bba79de99f79b670f62958e917c81ecc04c6e7 0	internal/vcs/git/files.go
		line := sc.Text()
		substr := line

		f1End := strings.IndexRune(substr, ' ')
		if f1End == -1 {
			return errors.New("field 1 not found")
		}
		status := substr[:f1End]
		if len(status) != 1 {
			return fmt.Errorf("expected status field value have a length of 1, got: %q", status)
		}
		o.Status = status[0]

		if skipUntracked && o.Status == ObjectStatusUntracked {
			continue
		}

		if len(substr) < f1End-1 {
			return fmt.Errorf("line has 1 field, expecting 2 or 5: %q", line)
		}
		substr = substr[f1End+1:]

		// objectmodes: https://git-scm.com/book/en/v2/Git-Internals-Git-Objects
		// > In this case, you’re specifying a mode of 100644, which
		// > means it’s a normal file. Other options are 100755, which
		// > means it’s an executable file; and 120000, which specifies a
		// > symbolic link. The mode is taken from normal UNIX modes but
		// > is much less flexible — these three modes are the only ones
		// > that are valid for files (blobs) in Git (although other
		// > modes are used for directories and submodules).
		var mode string
		f2End := strings.IndexRune(substr, ' ')
		if f2End == -1 {
			mode = substr
		} else {
			mode = substr[:f2End]
		}

		modeUint, err := strconv.ParseUint(mode, 8, 32)
		if err != nil {
			return fmt.Errorf("could not parse mode %q from substr: %q, line %q: %w", mode, substr, line, err)
		}
		o.Mode = Mode(modeUint)

		if f2End == -1 || len(substr) < f2End-1 {
			ch <- &o
			continue
		}
		substr = substr[f2End+1:]

		f3End := strings.IndexRune(substr, ' ')
		if f3End == -1 {
			return fmt.Errorf("line has 4 fields, expecting 2 or 5 fields: %q", line)
		}
		o.Name = substr[:f3End]

		if len(substr) < f3End-1 {
			return fmt.Errorf("line has 4 fields, expecting 2 or 5: %q", line)
		}
		substr = substr[f3End+1:]
		// we skip field 4 (stageNr, we don't need it)

		f5Start := strings.IndexRune(substr, '\t')
		if f5Start == -1 {
			return fmt.Errorf("line is missing '\t' character: %q", line)
		}
		o.RelPath = substr[f5Start+1:]

		ch <- &o
	}

	return sc.Err()
}
