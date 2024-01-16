package term

import (
	"fmt"
	"io"
	"sync"

	"github.com/fatih/color"

	"github.com/simplesurance/baur/v3/pkg/baur"
)

const separator = "------------------------------------------------------------------------------"

var ErrorPrefix = color.New(color.FgRed).Sprint("ERROR:")

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

// ErrPrintln prints an error with an optional message.
// The method prints the error in the format: errorPrefix msg: err
func (s *Stream) ErrPrintln(err error, msg ...interface{}) {
	if len(msg) == 0 {
		s.Println(ErrorPrefix, err)
		return
	}

	wholeMsg := fmt.Sprint(msg...)
	s.Printf("%s %s: %s\n", ErrorPrefix, wholeMsg, err)
}

// ErrPrintf prints an error with an optional printf-style message.
// The method prints the error in the format: errorPrefix msg: err
func (s *Stream) ErrPrintf(err error, format string, a ...interface{}) {
	s.ErrPrintln(err, fmt.Sprintf(format, a...))
}

// PrintErrln prints as message that is prefixed with "ERROR: "
func (s *Stream) PrintErrln(msg ...interface{}) {
	s.Println(ErrorPrefix, fmt.Sprint(msg...))
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
