package term

import (
	"fmt"
	"io"
	"strings"
	"sync"

	"github.com/fatih/color"

	"github.com/simplesurance/baur/v5/pkg/baur"
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

func (s *Stream) Printf(format string, a ...any) {
	s.lock.Lock()
	defer s.lock.Unlock()

	fmt.Fprintf(s.stream, format, a...)
}

func (s *Stream) Println(a ...any) {
	s.lock.Lock()
	defer s.lock.Unlock()

	fmt.Fprintln(s.stream, a...)
}

// TaskPrintf prints a message that is prefixed with '<TASK-NAME>: '
func (s *Stream) TaskPrintf(task *baur.Task, format string, a ...any) {
	prefix := Highlight(fmt.Sprintf("%s: ", task))

	s.Printf(prefix+format, a...)
}

// ErrPrintln prints an error with an optional message.
// The method prints the error in the format: errorPrefix msg: err
func (s *Stream) ErrPrintln(err error, msg ...any) {
	if len(msg) == 0 {
		s.Println(ErrorPrefix, err)
		return
	}

	wholeMsg := fmt.Sprint(msg...)
	s.Printf("%s %s: %s\n", ErrorPrefix, wholeMsg, err)
}

// ErrPrintf prints an error with an optional printf-style message.
// The method prints the error in the format: errorPrefix msg: err
func (s *Stream) ErrPrintf(err error, format string, a ...any) {
	s.ErrPrintln(err, fmt.Sprintf(strings.TrimSuffix(format, "\n"), a...))
}

// PrintErrln prints as message that is prefixed with "ERROR: "
func (s *Stream) PrintErrln(msg ...any) {
	s.Println(ErrorPrefix, fmt.Sprint(msg...))
}

// PrintErrf prints as message that is prefixed with "ERROR: "
func (s *Stream) PrintErrf(format string, a ...any) {
	s.Printf(ErrorPrefix+" "+format, a...)
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
