package terminal

import (
	"fmt"
	"io"
	"sync"

	"github.com/simplesurance/baur"
)

const separator = "------------------------------------------------------------------------------"

type OutputStreams struct {
	Stdout *Stream
	Stderr *Stream
}

type Stream struct {
	stream io.WriteCloser
	lock   sync.Mutex
}

func NewStream(out io.WriteCloser) *Stream {
	return &Stream{stream: out}
}

func (s *Stream) Printf(format string, a ...interface{}) {
	s.lock.Lock()
	defer s.lock.Unlock()

	fmt.Fprintf(s.stream, format, a...)
}

func (s *Stream) Println(a ...interface{}) {
	s.lock.Lock()
	defer s.lock.Unlock()

	fmt.Fprintln(s.stream, a...)
}

func (s *Stream) TaskPrintf(task *baur.Task, format string, a ...interface{}) {
	prefix := Highlight(fmt.Sprintf("%s: ", task))

	s.Printf(prefix+format, a...)
}

// PrintSep prints a separator line
func (s *Stream) PrintSep() {
	fmt.Fprintln(s.stream, separator)
}
