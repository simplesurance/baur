package term

import (
	"fmt"
	"io"
	"sync"

	"github.com/simplesurance/baur/v1/pkg/baur"
)

const separator = "------------------------------------------------------------------------------"

// Stream is a concurrency-safe output for term.messages.
type Stream struct {
	stream io.Writer
	lock   sync.Mutex
}

func NewStream(out io.Writer) *Stream {
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

// TaskPrintf prints a message that is prefixed with '<TASK-NAME>: '
func (s *Stream) TaskPrintf(task *baur.Task, format string, a ...interface{}) {
	prefix := Highlight(fmt.Sprintf("%s: ", task))

	s.Printf(prefix+format, a...)
}

// PrintSep prints a separator line
func (s *Stream) PrintSep() {
	fmt.Fprintln(s.stream, separator)
}

func (s *Stream) Write(p []byte) (n int, err error) {
	s.lock.Lock()
	defer s.lock.Unlock()

	return s.stream.Write(p)
}
